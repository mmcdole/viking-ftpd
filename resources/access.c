/*
 * FILE
 *     access.c
 *
 * DESCRIPTION
 *     access granting and verification daemon
 *
 * COPYRIGHT
 *     Copyright (C) 1995-2010 by Kris Van Hees, Belgium.  All rights reserved.
 *                   2006-2010 (modified) by Arvid Berdahl, Norway.
 */

#include <dgd.h>
#include <mudlib.h>
#include <levels.h>
#include <options.h>
#include <type.h>
#include <access.h>

/*
 * The (in)famous access map.
 *
 * While being called an 'access map', it is actually a collection  of  'access
 * trees', each representing a subset of the directory hierarchy  of  the  MUD.
 * In essence, there should be a tree for every player in the  MUD.   For  very
 * obvious practical reasons, various optimisations are used to avoid storing a
 * large collection of almost identical trees.  Each tree is stored as a nested
 * collection of mappings, using the following pseudo-syntax:
 *
 *  <tree>      ::= <subtree>
 *  <subtree>   ::= <access>
 *            | ([ <nodes> ])
 *  <nodes>     ::= <node>
 *            | <nodes>, <node>
 *  <node>      ::= ".": <access>
 *            | "*": <access>
 *            | <name>: <subtree>
 *  <access>    ::= -1          (REVOKED)
 *            | 1               (READ)
 *            | 2               (GRANT_READ)
 *            | 3               (WRITE)
 *            | 4               (GRANT_WRITE)
 *            | 5               (GRANT_GRANT)
 *  <name>      ::= String
 *
 * A "." node defines a specific access level for the subtree it belongs to.
 *
 * A "*" node defines a default access level for all nodes at this branch point
 * (and below), unless a specific access level is defined for a  node  at  this
 * branch point or lower.
 *
 * A named node defines the access level for itself and  the  default  for  all
 * nodes below it.
 *
 * Example:
 *
 *  (1) "": ([
 *  (2)        ".": READ,
 *  (3)        "*": READ,
 *  (4)        "data":  REVOKED,
 *  (5)        "log":   WRITE,
 *  (6)        "players": ([
 *  (7)              ".":   READ,
 *  (8)                  "*":   REVOKED,
 *  (9)                  "aedil":   GRANT_GRANT,
 * (10)                  "frogo": ([
 * (11)                          ".":   READ,
 * (12)                          "*":   REVOKED
 * (13)                           ])
 * (14)               ])
 * (15)     ])
 *
 * This means that the following paths have the specified access levels:
 *
 *  - /             READ
 *      The specific access level for the root is specified as READ  in
 *      line (2).
 *  - /characters           READ
 *      The default access for unspecified subtrees at the  root  level
 *      is READ as specified in line (3).
 *  - /data/notes           REVOKED
 *      Access is explicitly revoked for the /data subtree in line (4).
 *  - /log/driver           WRITE
 *      Write access is granted to /log and any directory below  (since
 *      access is specified for the subtree), in line (5).
 *  - /players          READ
 *      The contents of the /players directory is readable as specified
 *      with a local access level in line (7).
 *  - /players/aedil/com/access.c   GRANT_GRANT
 *      The /players/aedil subtree is given GRANT_GRANT access in  line
 *      (9).
 *  - /players/dios/workroom.c  REVOKED
 *      All unspecified subtrees of /players are revoked, as  specified
 *      in line (8).
 *  - /players/frogo        READ
 *      The contents of the /players/frogo directory can be listed  (as
 *      that operation uses the access level of /players/frogo  itself,
 *      and not the access levels of the contents  of  /players/frogo).
 *      Line (11) makes this possible.
 *  - /players/frogo/workroom.c REVOKED
 *      The actual contents of /players/frogo cannot be  read,  because
 *      of line (12).
 *
 * This shows how "." and "*" are very different.  The "." access level  really
 * affects the subtree root, whereas "*"  affects  the  primary  nodes  of  the
 * subtree.  The "." pseudo-node was introduced to handle the following special
 * access case:
 *
 *  0) User 'foo' has no access to any subdirectory of /players,  by  means
 *     of a '"*": REVOKED' element in the /players access specification.
 *  1) User 'foo' has read  access  to  /players/frogo,  reflected  in  the
 *     access tree with:
 *      "": ([ "players": ([ "*": REVOKED, "frogo": READ ]) ])
 *  2) Frogo decides to give user 'foo' write access to /players/com.  This
 *     would require an entry  '"frogo": ([ "*": READ, "com": WRITE ])'  to
 *     replace '"frogo": READ', but then the contents of /players/frogo  is
 *     no longer visible in ls, because the access level of  /players/frogo
 *     will be derived from the '"*": REVOKED' element to /players.
 *
 *     To resolve this, the access tree will be rewritten as:
 *      "": ([ "players": ([ "*": REVOKED,
 *                   "frogo": ([ ".": READ, "*": READ,
 *                           "com": WRITE" ]) ]) ])
 *
 *     This appropriately specifies that /players/frogo itself is indeed to
 *     be readable by 'foo'.
 *
 * The following trees can be present in the access map:
 *
 *  - The '*' tree is used as default access tree for all users.
 *
 *  - All trees for entities that start  with  a  capital  serve  as  group
 *    access trees for a specific subgroup of players.
 *
 *  - All trees for entities that start with a lower case letter are access
 *    trees for specific players in the MUD.
 *
 *  - Trees for players can have a special root node, named "?".  This node
 *    (if present) must have an array as value,  specifying  access  groups
 *    that the player belongs to.  Evaluating access for a  given  pathname
 *    uses a lazy evaluator algorithm, meaning that the first match is used
 *    as the access level.  First the player's access  tree  is  consulted.
 *    Then the group access trees, in order as they are  specified  in  the
 *    "?" node value (if present).  Finally, the "*" access tree is used.
 */

private mapping access_map_default; /* default access privileges */
mapping access_map;                 /* access database */

private string *admins;             /* hard-coded admins */
private string *fusers;             /* "fake" users */
private string *s_grps;             /* static groups */

string str_type(int type);

/*
 * FUNCTION
 *     private int save_db()
 *
 * DESCRIPTION
 *     save the access database
 */

private int save_db() {
    string  s;
    
    /*
     * If this does not work, make sure that:
     *     chmod u+w /local/viking/dgd/lib/dgd/sys/data !!
     */
    if (s = catch(save_object(DGD_ACCESS_DB))) {

        DGD_INIT->writek("PANIC: FAILED TO SAVE THE ACCESS DATABASE!\n");
        DGD_INIT->writek("PANIC: " + s + "\n");

        return 0; /* failure */
    }
    
    return 1; /* success */
}

/*
 * FUNCTION
 *     static void create()
 *
 * DESCRIPTION
 *     initialization routine
 */

static void create() {

    /* static groups */
    s_grps = ({ "Arch_full", "Arch_docs", "Arch_qc", "Arch_junior", "Arch_law", "Arch_web" });

    /* hard-coded admins */
    admins = ({ "moreldir", "kralk", "cryzeck" });

    /* "fake" users */
    fusers = ({ "*", "backbone", "root" });

    seteuid(getuid());

    /* Bootstrap access map to allow the daemon to read its savefile: */

    access_map =
    ([
        "*": ([
                 "*": REVOKED,
             ]),
        "root": ([
                    "dgd": ([
                               "sys": ([
                                          "data": READ,
                                      ]),
                           ]),
                ]),
    ]);

    /* Static, Default Access Privileges: 

           Note: Manual changes may be applied to this mapping here, 
                 and 'grant * default' will reset default access privileges
                 to this access mapping! */

    access_map_default =
    ([
	"*"         : READ,
	"characters": REVOKED,
	"d"         : ([
		          "*" : REVOKED,
			  "." : READ,
		      ]),
	"players"   : ([
			  "*" : REVOKED,
			  "." : READ,
		      ]),
	"data"      : REVOKED,
	"tmp"       : WRITE,
	"log"       : ([
                          "*" : READ,
                          "Driver" : REVOKED,
                          "old" : REVOKED,
               		]),
	"banish"    : REVOKED,
	"accounts"  : REVOKED,
	"dgd"       : REVOKED,
    ]);

    if (!restore_object(DGD_ACCESS_DB)) {
        
        access_map =
        ([
            /* default access privileges */
            "*"             : access_map_default,
                
            /* backbone privileges */
            "backbone"      : ([
		"*"         : WRITE,
            ]),

            /* root privileges */
            "root"          : ([
    	        "*"         : WRITE,
	    ]),

            /* default arch access privileges */
            "Arch_full"     : ([
                "*"         : GRANT_WRITE,
            ]),
            "Arch_junior"   : ([
                "d"         : WRITE,
                "players"   : WRITE,
            ]),

            /* default arch group access privileges */
            "Arch_docs"     : ([
                "help"      : WRITE,
                "doc"       : WRITE,
            ]),
            "Arch_law"      : ([
                "data"      : ([
                                  "Law"          : WRITE,
                              ]),
            ]),
            "Arch_qc"       : ([
                "data"      : ([
                                  "qc"           : WRITE,
                ]),
            ]),
            "Arch_web"      : ([
                "data"      : ([
                                  "www_docs"     : WRITE,
                              ]),
            ]),
        ]);

        save_db();
    }    
}

/*
 * FUNCTION
 *     string *query_all_groups()
 *
 * DESCRIPTION
 *     return array of all existing groups
 */

string *query_all_groups() {
    string *g, *u;
    int i;

    g = s_grps;

    for (u = keys(access_map), i = 0; i < sizeof(u); i++)
    {
        /* ignore fake users: */
        if (member_array(u[i], fusers) > -1)
	    continue;

	/* Groups have capital letters: */
	if (lower_case(u[i]) == u[i])
	    continue;

	/* Dont refer to static Groups twice: */
	if (member_array(u[i], g) > -1)
	    continue;

	g += ({ u[i] });
    }

    return g;
}

/*
 * FUNCTION
 *     private string *query_groups(string user)
 *
 * DESCRIPTION
 *     return array of groups 'user' is member of
 *
 * NOTE
 *     users with arch membership to specific groups
 *     in /secure/daemons/archgroupd.c will also
 *     have its corresponding access group added
 *     automagically (when logged on..)
 */

private string *query_groups(string user) {
    string *groups, *ugroupsb, *ugroupsa, *res, tmp;
    int i, l;

    /* optimize */
    if (!user || typeof(user) != T_STRING || member_array(user, fusers) > -1)
        return ({ });

    /* Groups have no groups, optimize */
    if (user != lower_case(user))
        return ({ });
    
    if (arrayp(groups = (string *)D_ARCHGROUP->query_data(user))) {
        for (i = 0; i < sizeof(groups); i++) {
            if (mapp(access_map[(tmp = "Arch_" + groups[i])]))
                groups[i] = tmp;
            else
                groups[i] = 0;
        }
        groups -= ({ 0 });
    }
    else
        groups  = ({   });
    
    if (!mapp(access_map[user]) || !arrayp(res = access_map[user]["?"]))
        res = ({ });
    
    ugroupsb = res; /* user groups before */
    ugroupsa = res; /* user groups after */
    
    /* specific arch privileges by level */
    if (mapp(access_map["Arch_full"]) && (l = (int)lookup_chardata(user, "level")) >= ARCHWIZARD)
        res += ({ "Arch_full" });
 
    else if (mapp(access_map["Arch_junior"]) && l >= JUNIOR_ARCH && l != ELDER)
        res += ({ "Arch_junior" });

    for (i = 0; i < sizeof(groups); i++) {
        if (member_array(groups[i], res) == -1) {
            
            /* if group does not exit, remove from users group list */
            if (!access_map[groups[i]])
                ugroupsa -= ({ groups[i] });
            else
                res += ({ groups[i] }); /* add valid group */
        }
    }
    
    /* if we encountered invalid user groups (deleted?), update players group list */
    if (sizeof(ugroupsa) != sizeof(ugroupsb)) {
        if (!sizeof(ugroupsa))
            ugroupsa = 0;
        access_map[user]["?"] = ugroupsa;
        save_db();
    }
        
    return res;
}

/*
 * FUNCTION
 *     int valid_users(string who, string user)
 *
 * DESCRIPTION
 *     determine whether 'who' is valid user for 'user'
 *
 *     function used by the grant.c cmd module.
 */

int valid_users(string who, string user) {
    
    /* is 'who' an access group? */
    if (who && who != lower_case(who))
        return 1;

    if (!D_FINGER->player_exists(who) &&
        (sizeof(admins & ({ user })) == 0 ||
         sizeof(fusers & ({ who })) == 0)) {
	    write("There is no such player on VikingMUD.\n");
	    return 0;
    }
	
	return 1;
}

/*
 * FUNCTION
 *     varargs string *_resolve(string path, string caller, string cwd, int flag)
 *
 * DESCRIPTION
 *     resolve a file path into an array of elements
 *
 * NOTE
 *     if 'flag' is true, "." in 'path' is allowed.
 */

varargs string *_resolve(string path, string caller, string cwd, int flag) {
    int     i, j, l, c, skip;
    string  *parts;
    object  ob;

    if (!path || !(l = strlen(path)))       /* invalid path */
        return 0;

    if (!caller &&
      (!(ob = DGD_RTE->this_player()) || !INTERACTIVEP(ob) ||
       !(caller = ob->query_real_name()))) {
        caller = "nobody";
        ob = 0;
    }

    switch (path[0]) {
        case '~':
            if (l > 1 && (c = path[1]) != '/') {
                path = ((c >= 'A' && c <= 'Z') ? "/d/"
                    : "/players/") + path[1 ..];
                break;
            } else if (caller != "root" && caller != "backbone")
                path = "/players/" + caller + path[1 ..];
            else
                path = path[2.. ];
            break;
        case '/':
            for (i = 0; i < l && path[i] == '/'; i++)
            ;

            path = path[i - 1 ..];
            break;
        default:
            if (cwd == "/")
                path = "/" + path;
            else {
                if (!cwd)
                     cwd = ob ? ob->query_path() : "";
                path = cwd + "/" + path;
            }
    }

#if 0
    if (!sscanf(path, "%*s/.%*s"))
        return explode(path, "/");
#endif
        
    for (i = j = l = sizeof(parts = explode(path, "/")); i; ) {
        switch (parts[--i]) {
            case "":
                continue;
            case "..":
                skip++;
                continue;
            case ".":
                /* allow removal of "." nodes by granting no-access */
                if (!flag) 
                    continue;
            default:
                if (skip) {
                    skip--;
                    continue;
                }
                parts[--j] = parts[i];
        }
    }

    return parts[j ..];
}

/* 
 * FUNCTION
 *     varargs string resolve(string path, string caller, string cwd)
 *
 * DESCRIPTION
 *     resolve a file path
 */

varargs string resolve(string path, string caller, string cwd) {
    string  *parts;

    return (parts = _resolve(path, caller, cwd))
        ? "/" + implode(parts, "/") : 0;
}

/*
 * FUNCTION
 *     private int eval_map(string part, mapping *list, int idx, int dfl, int final)
 *
 * DESCRIPTION
 *     evaluate an element against an access map
 */

private int eval_map(string part, mapping *list, int idx, int dfl, int final) {
    int     acc;
    mapping map;
    mixed   v;

    if (map = list[idx]) {            /* map not exhausted yet */
        if (!(acc = map["*"]))
            acc = dfl;

        if (v = map[part]) {          /* part found in map */
            if (typeof(v) == T_INT) { /* leaf in the map */
                acc = v;
                list[idx] = 0;        /* end of this map */
            } else {                  /* branch in the map */
                int acc2;
                if (!dfl)
                    acc = v["*"];
                else if(final && (acc2 = v["."]))   /* obey final "." control */
                    acc = acc2;
                else if(!final && (acc2 = v["*"]))  /* allow to inherit from * if not final */
                    acc = acc2;
                else
                    acc = dfl;
                list[idx] = v;
            }
        } else                        /* part not found in map */
            list[idx] = 0;            /* end of this map */
    } else
        acc = dfl;

    return acc;
}

/*
 * FUNCTION
 *     private mapping subtract_map(mapping map, mixed exclude)
 *
 * DESCRIPTION
 *     return a new mapping with elements from 'map' excluded
 *     keys defined in 'exclude'
 *
 *     function used by the local functions:
 *         'show_access' and 'get_access'
 */

private mapping subtract_map(mapping map, mixed exclude) {
    mapping res;
    string *k;
    int i;
    
    exclude = (arrayp(exclude) ? exclude : ({ exclude }));
    
    for (res = ([ ]), k = keys(map), i = 0; i < sizeof(k); i++) {
        if (member_array(k[i], exclude) > -1)
            continue;
        res[k[i]] = map[k[i]];
    }
    
    return res;
        
}

/*
 * FUNCTION
 *     private mixed get_access_maps(string user)
 *
 * DESCRIPTION
 *     collect the list of access maps that is available
 *     and needs to be considered for 'user'.
 *
 * NOTE
 *     function returns array of array-pairs, where each
 *     index holds ({ string map_name, mapping map }).
 */

private mixed get_access_maps(string user) {
    string *groups;
    mapping map;
    mixed maps;
    int i, s;

    s = sizeof(groups = query_groups(user));

    if ((map = access_map[user])) {   /*This if sentence has been split in two. the second part is the else if(s) - 12022021 -Moreldir */

        if (s) {
            /* user, groups, and default access map */
            maps = ::allocate(s + 2);

            maps[0] = ({ user, subtract_map((map = (mapp(map) ? map : ([ ]))), "?") });
            
            for (i = 0; i < s; i++)
                maps[i + 1] = ({ groups[i], access_map[groups[i]] });
            
            maps[s + 1] = ({ "*", access_map["*"] });
        }
        else {
            /* user, and default access map */
            maps = ({ ({ user, map }), ({ "*", access_map["*"] }) });
        }
    }
    else if(s) {  /* /* Extracted this from the previous generic if that created issues for arches * - 12022021 -Moreldir */
        /* groups and default */
            maps = ::allocate(s+1);
            for (i = 0; i < s; i++)
              maps[i] = ({ groups[i], access_map[groups[i]] });
            
            maps[s] = ({ "*", access_map["*"] });
    }
    else {
        /* default access map only */
        maps = ({ ({ "*", access_map["*"] }) });
    }
        
    return maps;
}

/*
 * FUNCTION
 *     private mixed _get_access(string path, string user, varargs mapping *maps)
 *
 * DESCRIPTION
 *     check what access type 'user' has in 'path', by evaluating
 *     access maps in array 'maps'
 *
 * NOTE
 *     if not last argument is given, all available access maps 
 *     for 'user' is evaluated.
 *
 *     function will return an array: ({ int access, string access_map }),
 *     where 'access' is the final access and 'access_map' is the name of
 *     the map granting the access.
 *
 *     for /d/<domain>/open/ and /players/<wiz>/open/, 'access_map' will be
 *     set to '!' which means: ruled access.
 */

private mixed _get_access(string path, string user, varargs mixed maps) {
    mapping map, *_maps;
    int i, j, sz, mapc;
    string *parts;
    int *dfls;
    object pl;

    /* allow character files to be saved by the player _only_ ! */
    if (sscanf(object_name(previous_object()), I_PLAYER + "#%*d") &&
        sscanf(path, "/characters/%*s/" + user))
        return ({ WRITE, "!" });

    /* collect the list of access maps that needs to be considered. */
    maps = (arrayp(maps) ? maps : get_access_maps(user));

    /* populate the list of default / privileges. */
    mapc = sizeof(maps -= ({ 0 }));
    dfls = ::allocate(mapc);

    for (_maps = ({ }), i = 0; i < mapc; i++) {
        dfls[i] = maps[i][1]["*"];
        _maps += ({ maps[i][1] });
    }

    /*
     * Evaluate the entire path against all the access maps (lazy evaluation).
     * Any valid access level (!= 0) is sufficient to abort the  current  part,
     * and move on to the next (earlier maps override later maps).
     */

    for (i = 0, sz = sizeof(parts = _resolve(path)); i < sz; i++) {
        for (j = 0; j < mapc; j++) {
            if ((dfls[j] = eval_map(parts[i], _maps, j, dfls[j], (i == sz - 1))) != 0)
                break;
        }
    }
    
    /*
     * At this point, 'j' indicates which access map provided the final  access
     * level, while 'dfls[j]' provides the actual access level.
     * Now it's time to deal with the second set of special  cases:  unless  an
     * explicit access level is set for in the user specific map (j == 0), full
     * access should be granted for a wizard's own  directory,  and
     * read access should be  granted  for  any  /d/./open  or  /players/./open
     * directory.  Note that the check for a userspecific map (j == 0) is  only
     * valid when there actually is a user specific map (mapc > 1).
     */

    if (sz >= 2 &&
       (parts[0] == "d" || parts[0] == "players") &&
       (j != 0 || mapc == 1)) {
        if (parts[1] == user)
            return ({ GRANT_GRANT, "!" });
        else if (sz >= 3 && parts[2] == "open")
            return ({ READ, "!" });
    }

    return ({ dfls[j], maps[j][0] });
}

/*
 * FUNCTION
 *     private int get_access(string path, string user)
 *
 * DESCRIPTION
 *     same as _get_access, except it simply returns what access
 *     'user' has in 'path'.
 */

private int get_access(string path, string user) {
    return (int)_get_access(path, user)[0];
}

/*
 * FUNCTION
 *     string valid_access(string path, string user, int acctype)
 *
 * DESCRIPTION
 *     check whether a user has a particular access to a path
 */

string valid_access(string path, string user, int acctype) {
    int acc;

    acc = get_access(path, user);

    /* If the access is denied, we log it! */
    if (acc < acctype) {
	string log;

	log = "'" + user + "' requests '" + str_type(acctype) + "' to '" +
	      path + "' with access '" + str_type(acc) + "'";
	catch(log = DGD_INIT->format_log_message(log, call_trace(), 0, 0)) ;

	log_file("/data/log/INVALID_ACCESS", log+"\n") ;
    }

    return acc >= acctype ? path : 0;
}

/*
 * FUNCTION
 *     int grant_access_group(string user, string group, int add)
 *
 * DESCRIPTION
 *     grant 'user' access to an access group
 * 
 * NOTE
 *     if 'add' is negative, 'user' will be removed from the
 *     specific access 'group'.
 */

int grant_access_group(string user, string group, int add) {
    mapping map;
    object ply;
    string *g;
    
    if (!(ply = this_player(1)))
        return -1;

    /* fake users cant be in groups: */
    if (member_array(user, fusers) > -1)
        return -6;

    /* not allowed to add Group to another Group */
    if (user != lower_case(user))
        return -5;

    g = query_groups(user);

    /* already member of this group? */
    if (member_array(group, g) > -1) {
        
        if (add)
            return -3;

	/* cant remove user with lvl < JR_ARCH from static groups: */
	if (member_array(group, s_grps) > -1 && (int)lookup_chardata(user, "level") >= JUNIOR_ARCH)
	    return -7;

        /* remove user from group */
        if (!sizeof((g -= ({ group, 0 }))))
            g = 0;
        
        if(!access_map[user])
            access_map[user] = ([ ]);    
        
        access_map[user]["?"] = g;
        
        /* if user had groups only and there are no more groups ...*/
        if (!g && !sizeof(keys(access_map[user])))
            access_map[user] = 0; /* remove user */
        
        save_db();
        
        return 1;
    }
    
    /* cannot remove users from a Group they are not member of */
    if (!add)
        return -4;

    /* cannot add users to a Group that does not exist */
    if (!mapp(access_map[group]))
        return -2;
        
    /* cannot add users to the static groups if you're not an archwizard or higher */
    if (member_array(group, s_grps) > -1 && ply->query_level() < ARCHWIZARD)
        return -8;
    
    /* add user to group */
    g += ({ group });

    if(!access_map[user])
        access_map[user] = ([ ]);    

    access_map[user]["?"] = g;

    save_db();
    
    return 2;
}

/*
 * FUNCTION
 *     int grant_access_default(string user)
 *
 * DESCRIPTION
 *     grant 'user' default access privileges.
 */

int grant_access_default(string user) {
    int *reqtype, l;
    object ply;
    
    if (!(ply = this_player(1)))
        return -1;

    /* does user exist? */
    if (!access_map[user])
        return 0;

    /* use lvl comparance for players */
    if (user == lower_case(user)) {
        
        /* special case for "fake users" (root, backbone, ... ) */
        if (member_array(user, fusers) > -1) {

        /* special case for "*" (default access map): */
	    if (user == "*") {

		access_map[user] = access_map_default;

		save_db();

		return 1;
	    }

            return -1;
	}
        
        /* arches++ can (with higher lvl than 'user') restore default access
         * privileges to other users with 1) lower level, or 2). themselves */
         
        if ((l = (int)ply->query_level()) >= ARCHWIZARD &&
            (l > (int)lookup_chardata(user, "level") ||
            (string)ply->query_real_name() == user)) {

            log_file("/data/log/GRANT",sprintf("%s reset %s's access privileges to default. Previous access: %O.\n",ply->query_real_name(),user,access_map[user]));
            
            access_map[user] = 0;
            
            save_db();
            
            return 1;
            
        }
        
        return -1;
    }
    
    /* arch wizards++ can restore group access to default */
    if ((l = (int)ply->query_level()) < ARCHWIZARD)
        return -1;

    log_file("/data/log/GRANT",sprintf("%s reset %s's access privileges to default. Previous access: %O.\n",ply->query_real_name(),user,access_map[user]));
    
    access_map[user] = 0;
    
    save_db();
    
    return 1;
}

/*
 * FUNCTION
 *     string str_type(int type)
 *
 * DESCRIPTION
 *     convert from int 'type' to description
 *
 * NOTE
 *     used by the grant cmd module, so cannot be
 *     private / static !!
 */

string str_type(int type) {
    switch (type) {
        case NO_ACCESS:
            return "no-access";
        case REVOKED:
            return "revoked";
	case READ:
	    return "read";
	case GRANT_READ:
	    return "grant-read";
	case WRITE:
	    return "write";
	case GRANT_WRITE:
	    return "grant-write";
	case GRANT_GRANT:
	    return "grant";
	default:
	    return 0;
    }
}

/*
 * FUNCTION
 *     private void log_grant(object user_granting, 
 *                            string user_target, 
 *                            string path, 
 *                            int acctype)
 *
 * DESCRIPTION
 *     log grant events to the appropriate homedir/log/
 *     for <user_granting> and <user_target> 
 *     / ACCESS_GRANTED.DGD-log
 */

private void log_grant(object user_granting, 
                       string user_target,
                       string path,
                       int acctype) {
    string path_log, log;
    
    if (user_target != lower_case(user_target))
        return; /* no logging for Groups */

    path_log = "/players/" + user_target + "/log";
    
    log = capitalize((string)user_granting->query_real_name()) + "(" +
          (int)user_granting->query_level() + ")";
    
    if (acctype == NO_ACCESS)
        log += " removed '" + user_target + "'s" + 
               " access to path: " + path + "\n";
    else
        log += " granted '" + user_target + "' " + str_type(acctype) +
               " access to path: " + path + "\n";

    /* log to /data/log/GRANT : */
    log_file("/data/log/GRANT", log);

    /* log to target user */
    
    if (file_size(path_log) != -2)
        return; /* directory does not exist */
    
    log_file(path_log + "/ACCESS_GRANTED", log);

    /* log to granting user */
    if ((string)user_granting->query_real_name() != user_target) {
        
        path_log = "/players/" + (string)user_granting->query_real_name() + "/log";
        
        if (file_size(path_log) != -2)
            return; /* directory does not exist */
        
        log_file(path_log + "/ACCESS_GRANTED", log);
    }
}

/*
 * FUNCTION
 *     int grant_access(string path, string user, int acctype)
 *
 * DESCRIPTION
 *     grant a particular access on a path for a user or Group
 */

int grant_access(string path, string user, int acctype) {
    int *reqtype, i, sz;
    string *parts;
    mixed map, v;
    object ply;
    
    if (!(ply = this_player(1)))
        return -1;
    
    /* verify that 'user' can grant this access type */
    switch (acctype) {
        case NO_ACCESS:
            reqtype = ({ GRANT_READ, GRANT_WRITE, GRANT_GRANT });
            break;
        case REVOKED:
            reqtype = ({ GRANT_READ, GRANT_WRITE, GRANT_GRANT });
            break;
        case READ:
            reqtype = ({ GRANT_READ, GRANT_WRITE, GRANT_GRANT });
            break;
        case GRANT_READ:
            reqtype = ({ GRANT_WRITE, GRANT_GRANT });
            break;
        case WRITE:
            reqtype = ({ GRANT_WRITE, GRANT_GRANT });
            break;
        case GRANT_WRITE:
            reqtype = ({ GRANT_GRANT });
            break;
        case GRANT_GRANT:
            reqtype = ({ GRANT_GRANT });
            break;
    }

    /* 'ply' has the right to grant this access? */
    if (sizeof(reqtype & ({ get_access(path, geteuid(ply)) })) == 0) {

        /* hard-coded admins still have access to proceed */
        if (member_array(geteuid(ply), admins) == -1)
            return -1;
    }

    /* special case for acctype == NO_ACCESS where we _remove_ a node */
    if (acctype == NO_ACCESS) {
    
        /* does 'user' have any access to 'path' (in personal access map)? */
        for (map = access_map[user], i = 0, sz = sizeof(parts = _resolve(path, 0, 0, 1)) - 1; i < sz; i++) {
            
            if (!mapp(map) || !mapp(v = map[parts[i]])) {
                map = 0;
                break;
            }
            
            map = v;
            
        }

        if (!map || !map[parts[i]])
            return 0;
        
        /* remove... */
        for (map = access_map[user], i = 0; i < sz; i++)
             map = map[parts[i]];
        
        map[parts[i]] = 0; /* ...node from branch */
        
        /* if branch holds only one node, convert from mapping to integer */
        if (sz >= 2 && mapp(map) && member_array(sizeof(keys(map)), ({ 1, 2 })) > -1 && (map["*"] || map["."])) {
            
            map["."] = 0;
            
            if (sizeof(keys(map)) == 1) {
                string k;
                
                for (map = access_map[user], k = "*", i = 0; i < sz - 1; i++)
                     map = map[parts[i]];
                
                if (!map[parts[i]][k])
                     k = map_indices(map)[0];
                
                map[parts[i]] = map[parts[i++]][k];
            }
        }
        
        /* if branch holds no more nodes... */
        if (!sizeof(keys(map))) {
            for (map = access_map[user], i = 0; i < sz - 2; i++)
                 map = map[parts[i]];
            map[parts[i]] = 0; /* ...remove branch */
        }
              
        
        /* if user has no more access nodes... */
        if (!sizeof(keys(access_map[user])))
            access_map[user] = 0; /* ...remove users access map */
            
        save_db();
        
        log_grant(ply, user, path, acctype);
        
        return (access_map[user] ? 1 : 2);
    }

    /* already have 'acctype' to 'path'? */
    else if (get_access(path, user) == acctype) { 
        
        /* Groups are allowed to be added access, so we need a special case
         * since e.g /d/Elandar/ will always have GRANT_GRANT to its own root */
        
        if (user != lower_case(user)) {

            /* does Group 'user' have any access to 'path' (in personal access map)? */
            for (map = access_map[user], i = 0, sz = sizeof(parts = _resolve(path, 0, 0, 1)) - 1; i < sz; i++) {
            
                if (!mapp(map) || !mapp(v = map[parts[i]])) {
                    map = 0;
                    break;
                }
            
                map = v;
            
            }

            if (map && typeof(v = map[parts[i]]) == T_INT && (int)v == acctype)
                return 0; /* Group already have this access */
        }
        
        else
            return 0;
    }

    /* adding a new access group? */
    if (ply && user != lower_case(user) && !mapp(access_map[user]))
        message("", "You create a new access group: '" + user + "'.\n", ply);

    /* initialize and validate 'user's access 'map': */
    if(!(map = access_map[user]))
         map = access_map[user] = ([ ]);

    for (i = 0, sz = sizeof(parts = _resolve(path)) - 1; i < sz; i++) {
	    
        if (intp(v = map[parts[i]])) {
	        
	        map = map[parts[i++]] = v ? ([ ".": v, "*": v, ]) : ([ ]);

	        while (i < sz)
                map = map[parts[i++]] = ([ ]);

            break;
	    }

        map = v;
    }
    
    /* now add the new access privileges...
     *
     * if we overwrite an existing node and acctype equals an existing
     * '*' global access type, remove the node and let the '*' rule. */

    if (typeof(v = map["*"]) == T_INT && v == acctype) {
        
        /* first remove our branch and let '*' grant acctype */

        map[parts[i]] = 0; 
            
        /* if remaining subtree has only one node left, convert from
         * mapping to a 'node-pair' NODE:ACCTYPE .. */
        
        if (sizeof(keys(map)) == 2 && map["*"] && (int)map["*"] == (int)map["."])
            map["."] = 0;
           
        if (sz > 0 && sizeof(keys(map)) == 1) {

            for(map = access_map[user], i = 0; i < sz - 1; i++)
                map = map[parts[i]];

            map[parts[i]] = acctype;
        }
    }

    else if (parts[i] == "*") {
        
        string *k;
        mixed n;
        int z;
        
        /* if the new node is a global '*' entry, and there are already
         * nodes at this level with equal acctype and these are of T_INT,
         * they are now obsolete and can be removed .. 
         *
         * example: bambi have access: ([ "players" : WRITE, ]) , and then
         *          'grant bambi write to /' would result in :
         *                             ([ "players" : WRITE, "*" : WRITE, ]),
         * => obviously the : ([ "players" : WRITE ]) is now obsolete ! */
            
        for (k = (string *)keys(map), z = 0; z < sizeof(k); z++) {
            if (typeof(n = map[k[z]]) == T_INT && (int)n == acctype)
                map[k[z]] = 0;   /* remove obsolete node */
        }
            
        map[parts[i]] = acctype; /* set (the new) nodes acctype */
    }

    else
        map[parts[i]] = acctype; /* set (the new) nodes acctype */

    log_grant(ply, user, path, acctype);

    save_db();

    return 1; /* success */        
}

/*
 * FUNCTION
 *     private string list_perm(int acctype)
 *
 * DESCRIPTION
 *     print a particular access type in human readable form
 */

private string list_perm(int acctype) {
    switch (acctype) {
        case REVOKED:
            return "(revoked)    ";
        case READ:
            return "(read)       ";
        case GRANT_READ:
            return "(grant read) ";
        case WRITE:
            return "(write)      ";
        case GRANT_WRITE:
            return "(grant write)";
        case GRANT_GRANT:
            return "(grant)      ";
    }
}

/*
 * FUNCTION
 *     private void bwrite(string pre, string path, string type)
 *
 * FUNCTION
 *     internal "bambi-write" func. used by the 'list_access' func.
 */

private void bwrite(string pre, string path, string type) {
    string tmp;
    int w, l;

    w = (this_player() ? (int)this_player()->query_width() : 80);    
    l = strlen(pre) + strlen(path) + 3;

    printf("%s%s   %'.'*s %s\n", pre, path, w - l - 18, "", type);
}

/*
 * FUNCTION
 *     private mapping list_access(string base, mapping amap, mapping dmap, string owner, int combine)
 *
 * DESCRIPTION
 *     list which access exists in a combined access entry
 */

private mapping list_access(string base, mapping amap, mapping dmap, string owner, int combine) {
    string dir, pre;
    string *dirs;
    int i, sz;
    mixed prv;

    if(!mapp(amap))
        amap = ([]);

    if(!mapp(dmap))
        dmap = ([]);

    pre = (owner ? "  " + owner + " " : "    ");

    for (sz = sizeof(dirs = keys(amap + dmap)); i < sz; ) {
    
        if (prv = amap[dir = dirs[i++]])
            if (dir == ".") {
                if (combine && i > 0 && typeof(amap[dirs[i - 1]]) == T_INT &&
                    typeof(prv) == T_INT && (int)amap[dirs[i - 1]] == (int)prv)
                    continue;
                bwrite(pre, base + ".", list_perm(prv));
            }
            else if (dir == "*")
                bwrite(pre, base, list_perm(prv));
            else if (typeof(prv) == T_INT)
                bwrite(pre, base + dir, list_perm(prv));
            else {
                mixed   map;

                if (typeof(map = dmap[dir]) != T_MAPPING)
                    map = ([]);

                list_access(base + dir + "/", prv, map, owner, combine);
            }
        else { /* how is this even possible?! -- bambi */
            prv = dmap[dir];

            if (dir == ".")
                bwrite("  D ", base + ".", list_perm(prv));
            else if (dir == "*")
                bwrite("  D ", base, list_perm(prv));
            else if (typeof(prv) == T_INT)
                bwrite("  D ", base + dir, list_perm(prv));
            else
                list_access(base + dir + "/", ([]), prv, owner, combine);
        }
    }
}

/*
 * FUNCTION
 *     private void merge_maps(mapping dmap, mapping amap, int dflt)
 *
 * DESCRIPTION
 *     merge two access maps
 */

private void merge_maps(mapping dmap, mapping amap, int dflt) {
    int i, eltc, ndfl;
    string *elts;

    elts = map_indices(amap) - ({ "*" });
    eltc = sizeof(elts);

    if (!(ndfl = dmap["*"]))
        ndfl = dflt;

    for (i = 0; i < eltc; i++) {
        mixed v, w;

        if (v = dmap[elts[i]]) {
            if (typeof(v) == T_INT)
                continue;
            if (typeof(w = amap[elts[i]]) == T_INT) {
                if(!v["*"])
                    v["*"] = w;
            } 
            else
                merge_maps(v, w, ndfl);
        } 
        else if (dmap["*"]) {
            continue;
        } else if (typeof(w = amap[elts[i]]) == T_MAPPING) {
            dmap[elts[i]] = v = ([ ]);
            merge_maps(v, w, ndfl);
        } else
            dmap[elts[i]] = w;
    }

    if(!dmap["*"] && !dflt && amap["*"])
        dmap["*"] = amap["*"];
}

/*
 * FUNCTION
 *     int show_access(string user, int flag)
 *
 * DESCRIPTION
 *     show which accesses a user has
 *
 * ARGUMENTS
 *     if flag == 0 , the full combined access map of 'user' is displayed,
 *     and in order of priority - with highest priority listed first.
 *     (detailed)
 *
 *     if flag == 1 , only the effective access for 'user' is displayed,
 *     while overruled access with lower priority is ignored.
 *     (not detailed)
 *
 *     if flag == 2 , the access mapping is displayed,
 *     useful for debugging purposes.
 *     (as is).
 */

int show_access(string user, varargs int flag) {
    int i, nr, sz, mapc, w;
    string *list, utype, line;
    mapping map, dmap;
    mixed maps;

    w = (this_player() ? (int)this_player()->query_width() : 80);
    
    /* attempt make confusion less confusing .. */
    if (this_player(1) && 
        member_array(user, fusers) == -1 && 
        !access_map[user] &&
        sizeof(query_groups(user)) <= 1) {

        write("No such " + (user != lower_case(user) ? "Group" : "user") + " in the database.\n");
        if (user != lower_case(user)) {
            write("Arch wizards can create a group: grant <Group> <acctype> to <path>\n");
            return 1;
        }
        else if (!D_FINGER->player_exists(user))
            return 1;
        else
            write("\nBut valid character file found...\nDefault access privileges will be used:\n\n");
        user = "*";
    }

    /* no detailed access status for default access privileges */
    if (user == "*" && !flag)
        flag = 1;

    /* collect the list of access maps that needs to be considered. */
    maps = get_access_maps(user);

    mapc = sizeof(maps -= ({ 0 }));
    dmap = ([ ]);

    line = sprintf("%'-'*s\n", w, "");

    /* list effective access (combined): */
    if (flag == 1) {

        for (i = 0; i < mapc; i++)
            merge_maps(dmap, maps[i][1], 0);

        write("Access privileges (effective) for "+(user == lower_case(user) ? 
              "user" : "group")+": " + user + "\n" + line);
        list_access("/", dmap, 0, 0, flag);
	write(line);
    }

    /* list mapping (as is): */
    else if (flag == 2) {
	string res, *__res, __name;

	write("Access privileges (mappings - as is) for "+(user == lower_case(user) ?
	      "user" : "group")+": " + user + "\n" + line);

	for (i = (user == "*" ? 1 : 0); i < mapc; i++) {

	    res = swrite(maps[i][1]);

     	    res = replace_string(res, ": -1,", ": -1, /* (REVOKED) */");
	    res = replace_string(res, ": 1," , ":  1, /* (READ) */");
	    res = replace_string(res, ": 2," , ":  2, /* (GRANT_READ) */");
	    res = replace_string(res, ": 3," , ":  3, /* (WRITE) */");
	    res = replace_string(res, ": 4," , ":  4, /* (GRANT_WRITE) */");
	    res = replace_string(res, ": 5," , ":  5, /* (GRANT_GRANT) */");

	    __name = maps[i][0];

	    if (__name == user)
	        __name += " (" + (__name == lower_case(__name) 
			       ? (user == "*" ? "Default" : "Personal") 
			       : "Group"   ) + " Mapping)";
	    else if (__name == "*")
		__name += " (Default Privileges)";
	    else if (__name != lower_case(__name))
		__name += " (Group)";

	    printf("%s :\n%'-'*s\n", bold(__name), w - 20, "");

            /* format our mapping output to human readable: */
            for (__res = explode(res, "\n"), nr = 0; nr < sizeof(__res); nr++) {
		write("    " + __res[nr] + "\n");
	    }

	    if (i < mapc - 1)
	        write("\n");
	}

	write(line + "Listed in order of priority (earlier overrules later access mappings).\n");
    }     
    
    /* list detailed (all) access: */
    else {

        write("Access privileges (detailed) for "+(user == lower_case(user) ? 
              "user" : "group")+": " + user + "\n" + line);
        
        for (i = 0, nr = 1; i < mapc; i++) {
            
            if (!sizeof(keys(maps[i][1])))
                continue;
            
            if (maps[i][0] == "*")
                utype = "";
            else if (maps[i][0] == lower_case(maps[i][0]))
                utype = "user ";
            else
                utype = "group ";
            write(sprintf("  #%-2d-", nr++) + " Access granted for " + 
                  utype + "'" + maps[i][0] + "'" +
                  (maps[i][0] == "*" ? " (default privileges)" : "") + ":\n");
            list_access("/", maps[i][1], 0, 0, flag);
            if (i < mapc - 1)
                write("\n");
        }
        
        write(line + "Listed in order of priority (earlier access overrules later privileges).\n");

    }

    return 1;
}

/*
 * FUNCTION
 *     mixed **expand_path(string path, string user)
 *
 * DESCRIPTION
 *     expand a path (with possible wildcards)
 */
 
mixed **expand_path(string path, string user) {
    mixed **list, **res;
    int i, sz, hidden;
    string *parts;
    string pref;

    if (path == "/")
        return ({ ({ "/", -2, 0 }) });

    list = ({ ({ "", -2, 0 }) });
    pref = "";
    parts = explode(path, "/") - ({ "" });
    
    for (i = 0, sz = sizeof(parts) - 1; i < sz; i++) {
        string  part;

        part = parts[i];

        if (sscanf(part, "%*s*") || sscanf(part, "%*s?")) {
            mixed *data, **nlst;
            int j, sy;

            nlst = ({ });
            hidden = part[0] == '.';
        
            for (j = 0, sy = sizeof(list); j < sy; j++)
                if (data = get_dir_compat((path = list[j][0] + pref + "/") + part)) {
                    int k;

                    for (k = sizeof(data); k--; )
                        if (data[k][1] == -2 &&
                           (hidden || data[k][0][0] != '.'))
                            data[k][0] = path + data[k][0];
                        else
                            data[k] = 0;
                    nlst += data - ({ 0 });
                }

            list = nlst;
            pref = "";
        } else
            pref += "/" + part;
    }

    if (strlen(pref)) {
        for (i = sizeof(list); i--; )
            if (!stat(list[i][0] += pref))
                list[i] = 0;
            list -= ({ 0 });
    }

    seteuid(user);

    res = ({});
    pref = parts[sz];
    hidden = pref[0] == '.';

    for (i = 0, sz = sizeof(list); i < sz; i++) {
        mixed *data;

        if (data = get_dir_compat((path = list[i][0] + "/") + pref)) {
            int k;

            for (k = sizeof(data); k--; )
                if (hidden || data[k][0][0] != '.')
                    data[k][0] = path + data[k][0];
                else
                    data[k] = 0;

            res += data - ({ 0 });
        }
    }

    seteuid(getuid());

    return res;
}
 
