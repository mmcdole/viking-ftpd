# Player Authentication in VikingMUD

VikingMUD stores player data in character files which are organized in a directory structure. This document describes how player authentication works and the format of the character files that store credentials.

## Directory Structure

Character files are stored in subdirectories based on the first letter of the username:

```
characters/
  a/
    aedil.o
    alice.o
  b/
    bob.o
  f/
    frogo.o
```

Each character's data is stored in a `.o` file named after their username (e.g., `alice.o` for user "alice").

## File Format

Character files are stored in the [LPC object format](lpc_object_format.md). The file contains various fields describing the character's attributes and settings. These files are parsed using the `pkg/lpc/oparser` package which implements a non-strict LPC object parser.

Example character file:
```
name "drake"
cap_name "Drake"
password "tek4edTZE898g"
level 45
gender 1
Str 29
Int 29
Con 29
Dex 29
experience 990147374
...
```

For authentication purposes, the most important field is:

- `password`: Contains the character's password hash. Supported formats:
  - Legacy Unix `crypt(3)` (13-char DES hash)
  - Argon2id in PHC format: `$argon2id$v=19$m=...,t=...,p=...$<salt_b64>$<hash_b64>`

### Password Hashing

The daemon supports two hashing schemes during migration:

1) [DES-based Unix crypt(3)](https://en.wikipedia.org/wiki/Crypt_(C)) (legacy)

- Uses the first two characters of the password as the salt
- Produces a 13-character hash string using a modified version of DES
- The first two characters of the hash are the salt used
- Compatible with the standard Unix `crypt(3)` function from libc

Example hash format: `XXyyyyyyyyyyy` where:
- `XX`: The two-character salt
- `yyyyyyyyyyy`: The remaining 11 characters of the hash

Note: The traditional Unix crypt algorithm only uses the first 8 characters of the password. Any characters beyond the 8th position are ignored; kept for compatibility with existing character files.

2) Argon2id (modern)

- Store full PHC string, e.g.: `$argon2id$v=19$m=65536,t=2,p=1$MDEyMzQ1Njc4OWFiY2RlZg$<hash>`
- The server auto-detects Argon2id by the `$argon2id$` prefix

### Authentication Process

1. The FTP daemon receives login credentials (username and password)
2. The system locates the character file in `characters/[first_letter]/[username].o`
3. The file is parsed as an LPC object to extract the password hash
4. The provided password is verified against the stored hash:
   - If the hash starts with `$argon2id$`, verify using Argon2id with the stored parameters/salt
   - Otherwise, verify using legacy Unix crypt with the salt from the first two characters
   - Authentication succeeds only if verification passes
