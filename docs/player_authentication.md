# Player Authentication in VikingMUD

VikingMUD stores player data in [LPC object files](lpc_object_format.md) organized in a directory structure. This document describes how player authentication works and the format of the character files that store credentials.

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

Character files are stored in the LPC object format. The file contains various fields describing the character's attributes and settings. For authentication purposes, the most important field is:

- `password`: Contains the character's password hash using Unix crypt format

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

### Password Hashing

Passwords are hashed using the [DES-based Unix crypt(3)](https://en.wikipedia.org/wiki/Crypt_(C)) algorithm with the following characteristics:

- Uses the first two characters of the password as the salt
- Produces a 13-character hash string using a modified version of DES
- The first two characters of the hash are the salt used
- Compatible with the standard Unix `crypt(3)` function from libc

Example hash format: `XXyyyyyyyyyyy` where:
- `XX`: The two-character salt
- `yyyyyyyyyyy`: The remaining 11 characters of the hash

Note: The traditional Unix crypt algorithm only uses the first 8 characters of the password. Any characters beyond the 8th position are silently ignored during both hashing and verification. This is a limitation of the original algorithm and is preserved for compatibility with existing character files.

### Authentication Process

1. The FTP daemon receives login credentials (username and password)
2. The system locates the character file in `characters/[first_letter]/[username].o`
3. The file is parsed as an LPC object to extract the password hash
4. The provided password is verified against the stored hash:
   - The first two characters of the stored hash are used as the salt
   - The password is hashed using this salt
   - The resulting hash is compared with the stored hash
   - If they match, authentication succeeds
