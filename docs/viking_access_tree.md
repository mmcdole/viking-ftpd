# VikingMUD Access Tree

> **Copyright Notice**  
> Copyright (C) 1995-2010 by Kris Van Hees, Belgium. All rights reserved.  
> 2006-2010 (modified) by Arvid Berdahl, Norway.

The (in)famous access map.

While being called an 'access map', it is actually a collection of 'access trees', each representing a subset of the directory hierarchy of the MUD. In essence, there should be a tree for every player in the MUD. For very obvious practical reasons, various optimizations are used to avoid storing a large collection of almost identical trees. Each tree is stored as a nested collection of mappings, using the following pseudo-syntax:

```
<tree>      ::= <subtree>
<subtree>   ::= <access>
            | ([ <nodes> ])
<nodes>     ::= <node>
            | <nodes>, <node>
<node>      ::= ".": <access>
            | "*": <access>
            | <name>: <subtree>
<access>    ::= -1          (REVOKED)
            | 1               (READ)
            | 2               (GRANT_READ)
            | 3               (WRITE)
            | 4               (GRANT_WRITE)
            | 5               (GRANT_GRANT)
<name>      ::= String
```

A "`.`" node defines a specific access level for the subtree it belongs to.

A "`*`" node defines a default access level for all nodes at this branch point (and below), unless a specific access level is defined for a node at this branch point or lower.

A named node defines the access level for itself and the default for all nodes below it.

Example:

```
(1) "": ([
(2)        ".": READ,
(3)        "*": READ,
(4)        "data":  REVOKED,
(5)        "log":   WRITE,
(6)        "players": ([
(7)              ".":   READ,
(8)                  "*":   REVOKED,
(9)                  "aedil":   GRANT_GRANT,
(10)                  "frogo": ([
(11)                          ".":   READ,
(12)                          "*":   REVOKED
(13)                           ])
(14)               ])
(15)     ])
```

This means that the following paths have the specified access levels:

- `/`             READ  
  The specific access level for the root is specified as READ in line (2).
- `/characters`           READ  
  The default access for unspecified subtrees at the root level is READ as specified in line (3).
- `/data/notes`           REVOKED  
  Access is explicitly revoked for the /data subtree in line (4).
- `/log/driver`           WRITE  
  Write access is granted to /log and any directory below (since access is specified for the subtree), in line (5).
- `/players`          READ  
  The contents of the /players directory is readable as specified with a local access level in line (7).
- `/players/aedil/com/access.c`   GRANT_GRANT  
  The /players/aedil subtree is given GRANT_GRANT access in line (9).
- `/players/dios/workroom.c`  REVOKED  
  All unspecified subtrees of /players are revoked, as specified in line (8).
- `/players/frogo`        READ  
  The contents of the /players/frogo directory can be listed (as that operation uses the access level of /players/frogo itself, and not the access levels of the contents of /players/frogo). Line (11) makes this possible.
- `/players/frogo/workroom.c` REVOKED  
  The actual contents of /players/frogo cannot be read, because of line (12).

This shows how "`.`" and "`*`" are very different. The "`.`" access level really affects the subtree root, whereas "`*`" affects the primary nodes of the subtree. The "`.`" pseudo-node was introduced to handle the following special access case:

1. User 'foo' has no access to any subdirectory of /players, by means of a '"`*`": REVOKED' element in the /players access specification.
2. User 'foo' has read access to /players/frogo, reflected in the access tree with:
   ```
   "": ([ "players": ([ "*": REVOKED, "frogo": READ ]) ])
   ```
3. Frogo decides to give user 'foo' write access to /players/com. This would require an entry '"frogo": ([ "*": READ, "com": WRITE ])' to replace '"frogo": READ', but then the contents of /players/frogo is no longer visible in ls, because the access level of /players/frogo will be derived from the '"`*`": REVOKED' element to /players.

To resolve this, the access tree will be rewritten as:
```
"": ([ "players": ([ "*": REVOKED,
                "frogo": ([ ".": READ, "*": READ,
                        "com": WRITE" ]) ]) ])
```

This appropriately specifies that /players/frogo itself is indeed to be readable by 'foo'.

The following trees can be present in the access map:

- The "`*`" tree is used as default access tree for all users.

- All trees for entities that start with a capital serve as group access trees for a specific subgroup of players.

- All trees for entities that start with a lower case letter are access trees for specific players in the MUD.

- Trees for players can have a special root node, named "`?`". This node (if present) must have an array as value, specifying access groups that the player belongs to. Evaluating access for a given pathname uses a lazy evaluator algorithm, meaning that the first match is used as the access level. First the player's access tree is consulted. Then the group access trees, in order as they are specified in the "`?`" node value (if present). Finally, the "`*`" access tree is used.
