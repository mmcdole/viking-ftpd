# LPC Object Format Specification

This document specifies the format used for saving and restoring LPC objects, based on the DGD implementation.

## 1. File Format

### 1.1 General Structure
- The file consists of a sequence of variable definitions
- Each definition must be on a separate line
- Lines are terminated with a newline character (`\n`)
- Empty lines (containing no characters or just a newline) are ignored
- Lines starting with `#` are treated as comments and ignored

### 1.2 Line Format
```
<variable_name> <value>\n
```
- No leading whitespace allowed at start of line
- Exactly one space between variable name and value (no tabs or multiple spaces)
- No trailing whitespace after value
- Must end with newline
- Comments must also follow these rules (no leading whitespace before #)

### 1.3 Examples
```
name "bob"              # Valid: one space after name
age 25                 # Valid: one space after age
                      # Valid: empty line
# comment              # Valid: comment at start of line
   name "bob"          # Invalid: leading spaces
name    "bob"          # Invalid: multiple spaces
name "bob"   \n        # Invalid: trailing spaces
name\t"bob"            # Invalid: tab instead of space
   # comment          # Invalid: leading space before comment
```

## 2. Variable Names

### 2.1 Format
- Must start with a letter (a-z, A-Z) or underscore
- Subsequent characters can be:
  - Letters (a-z, A-Z)
  - Numbers (0-9)
  - Underscores (_)
- No length limit specified

### 2.2 Examples
```
valid_name
_name
name123
```

## 3. Value Types

### 3.1 Strings
- Enclosed in double quotes
- Must be terminated (no newlines allowed inside string)
- Supports the following escape sequences:
  - `\0` - null character
  - `\a` - bell (BEL)
  - `\b` - backspace
  - `\t` - tab
  - `\n` - newline
  - `\v` - vertical tab
  - `\f` - form feed
  - `\r` - carriage return
  - `\"` - double quote
  - `\\` - backslash
  - Any other character after backslash is taken literally (e.g. `\z` becomes `z`)
- Examples:
  ```
  "hello world"                    # Basic string
  "hello \"world\""                # Escaped quotes
  "C:\\path\\to\\file"            # Escaped backslashes
  "line1\nline2\tcolumn2"         # Newlines and tabs
  "alert\a\b\v\f\r"               # Control characters
  "hello\zworld"                  # Unknown escape (becomes 'hellozworld')
  ```

### 3.2 Numbers
#### Integers
- Optional negative sign
- Sequence of digits
- No leading zeros required
- Example: `-123`, `456`

#### Floats
- Optional negative sign
- Decimal part (required)
- Optional hexadecimal mantissa after `=` sign
- Format: `<decimal>=<hex>` or just `<decimal>`
- Examples: 
  - `3.14`
  - `-2.5`
  - `1.0=3ff0000000000000`  # The hex part represents the IEEE 754 bits

### 3.3 Arrays
- Enclosed in `({` and `})`
- Size is specified after opening delimiter, followed by `|`
- Elements are comma-separated
- Size must match number of elements
- Can contain nested arrays and mappings
- Examples:
  ```
  ({0|})                           # Empty array
  ({3|1,2,3})                     # Simple array
  ({3|"hello",42,3.14})          # Mixed types
  ({2|({2|1,2}),({2|3,4})})      # Nested arrays
  ({3|nil,1,nil})                # Array with nil values
  ```

### 3.4 Mappings
- Start with `([`
- Format: `([size|key1:value1,key2:value2,...])`
- Size indicates number of key-value pairs
- Key-value pairs separated by `:`
- Pairs separated by commas
- End with `])`
- Keys and values can be any valid value type including:
  - Strings
  - Numbers
  - Arrays
  - Other mappings
  - nil
- Mappings are sorted after parsing (implementation detail)
- Examples:
  ```
  (["name":"bob", "age":25])           # String keys
  ([123:456])                          # Number key
  ([([1:2]):([3:4])])                 # Mapping as key
  ([({"array"}):123])                 # Array as key
  ([nil:"value"])                     # nil as key
  ```

### 3.5 Nil
- The literal string "nil" (lowercase only)
- Represents null/undefined value

### 3.6 References
- Arrays: `#n` where n is the array index
- Mappings: `@n` where n is the mapping index
- Indices are zero-based
- Must refer to previously defined arrays/mappings

## 4. Examples

### 4.1 Complete Object Example
```
# User data
name "Bob"
age 25
scores ({3|90,85,95})
data (["city":"New York","active":1])
backup_scores #0
settings @1
```

### 4.2 Complex Nested Structures
```
users ({2|(["name":"alice","age":25]),(["name":"bob","age":30])})
groups (["admins":({3|1,2,3}),"users":({3|4,5,6})])
```

## 5. Error Conditions

The parser must detect and handle these error conditions:

### 5.1 Syntax Errors
- Invalid variable names
- Missing space after variable name
- Unterminated strings
- Unterminated arrays/mappings
- Invalid array/mapping references
- Missing newline at end of line

### 5.2 Type Errors
- Value type doesn't match variable type (if type checking enabled)
- Invalid reference indices
- Malformed numbers

## 6. Implementation Notes

### 6.1 Parsing Strategy
- Read entire file into memory
- Process line by line
- Track array/mapping references for later resolution
- Validate syntax before assigning values

### 6.2 Reference Handling
- Maintain counters for arrays and mappings
- Resolve references after initial parse
- Handle circular references appropriately

### 6.3 Type Checking
- Optional strict type checking
- Allow T_MIXED for flexible typing
- nil allowed for pointer types
- Array element types must match if specified

## 7. Compatibility Notes

This format is based on DGD's implementation and may differ from other LPMud drivers. Key compatibility points:

- Strict whitespace handling
- Specific array/mapping syntax
- Reference counting system
- Comment handling
