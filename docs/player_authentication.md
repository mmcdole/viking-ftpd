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

- `password`: Contains the character's password hash (Unix crypt or Argon2id format)

### Password Hashing

VikingMUD supports multiple password hashing algorithms with automatic detection:

#### Unix Crypt (Legacy)
The traditional hashing method using the [DES-based Unix crypt(3)](https://en.wikipedia.org/wiki/Crypt_(C)) algorithm:

- Uses the first two characters of the password as the salt
- Produces a 13-character hash string using a modified version of DES
- The first two characters of the hash are the salt used
- Compatible with the standard Unix `crypt(3)` function from libc
- Example format: `XXyyyyyyyyyyy` where `XX` is the salt and `yyyyyyyyyyy` is the hash

Note: The traditional Unix crypt algorithm only uses the first 8 characters of the password. Any characters beyond the 8th position are silently ignored during both hashing and verification.

#### Argon2id (Modern)
The recommended modern hashing algorithm for new character files:

- Uses the Argon2id variant, which provides resistance against both side-channel and GPU-based attacks
- Configurable memory usage, iteration count, and parallelism parameters
- Produces variable-length hashes with embedded salt and parameters
- Example format: `$argon2id$v=19$m=65536,t=3,p=4$salt$hash`

#### Automatic Detection
The FTP daemon automatically detects the hash type:
- Hashes starting with `$argon2id$` are processed as Argon2id
- 13-character hashes without `$` symbols are processed as Unix crypt
- This ensures backward compatibility while supporting modern security standards

### Authentication Process

1. The FTP daemon receives login credentials (username and password)
2. The system locates the character file in `characters/[first_letter]/[username].o`
3. The file is parsed as an LPC object to extract the password hash
4. The system automatically detects the hash format (Unix crypt or Argon2id)
5. The provided password is verified against the stored hash using the appropriate algorithm:
   - **Unix crypt**: Uses the first two characters as salt, hashes with DES, and compares
   - **Argon2id**: Parses parameters from the hash, re-computes with same settings, and compares
   - If they match, authentication succeeds
