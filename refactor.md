# Character Management Refactoring Plan

## Overview
We are refactoring the character management system in Viking FTP to improve separation of concerns, caching, and access control. This refactoring extracts character data management into a dedicated package that can be used by both authentication and authorization systems.

## Goals
1. Separate character data management from authentication logic
2. Implement efficient caching of character data
3. Provide a clean interface for both authentication and authorization systems
4. Maintain existing security and access control features
5. Make the codebase more maintainable and testable

## Package Structure

### New Package: `playerdata`
- **Purpose**: Central location for character data management
- **Key Components**:
  - `Character`: Data model for character information
  - `Repository`: Main interface for character data access with caching
  - `Source`: Interface for different storage backends
  - `FileSource`: Implementation for LPC object file storage
  - `MemorySource`: Implementation for testing

### Changes to Existing Packages

#### `authentication` Package
- Remove direct file handling
- Use `playerdata.Repository` for character lookups
- Focus purely on authentication logic
- Maintain password verification functionality

#### `authorization` Package (Planned)
- Remove direct file handling
- Use `playerdata.Repository` for character lookups
- Implement level-based group resolution
- Focus on authorization rules and group membership

## Detailed Changes

### 1. New Types and Interfaces

#### Character Model
```go
type Character struct {
    Username     string
    PasswordHash string
    Level        int
}
```

#### Source Interface
```go
type Source interface {
    LoadCharacter(username string) (*Character, error)
}
```

### 2. Repository Implementation
- Implements caching with configurable duration
- Thread-safe character data access
- Handles cache invalidation and refresh
- Provides methods:
  - `GetCharacter(username string) (*Character, error)`
  - `RefreshCharacter(username string) error`
  - `UserExists(username string) (bool, error)`

### 3. File Source Implementation
- Maintains existing LPC object file structure
- Handles character subdirectories (by first letter)
- Parses LPC object format
- Extracts password hash and level information

### 4. Authentication Updates
- Remove `CharacterFile` type
- Update `Authenticator` to use `playerdata.Repository`
- Remove duplicate caching logic
- Maintain existing authentication interface

### 5. Authorization Updates (Planned)
- Update group resolution to use `playerdata.Repository`
- Implement implicit level-based permissions
- Maintain existing authorization interface

## Testing Strategy
1. Unit tests for each component:
   - Repository caching behavior
   - File source LPC parsing
   - Memory source for testing
   - Updated authenticator tests
   - Authorization tests (planned)

2. Integration tests:
   - Character loading and caching
   - Authentication flow
   - Authorization flow (planned)

## Migration Steps
1. ✓ Create `playerdata` package
2. ✓ Implement core types and interfaces
3. ✓ Implement repository with caching
4. ✓ Implement file source
5. ✓ Update authentication package
6. □ Update authorization package
7. □ Add integration tests
8. □ Update main application to use new structure

## Benefits
1. **Maintainability**: Clear separation of concerns
2. **Performance**: Efficient caching of character data
3. **Testability**: Memory source for testing
4. **Reusability**: Single source of character data
5. **Security**: Maintained existing security model

## Risks and Mitigations
1. **Risk**: Breaking existing authentication
   - **Mitigation**: Comprehensive test coverage
   - **Mitigation**: Gradual rollout

2. **Risk**: Cache consistency issues
   - **Mitigation**: Clear cache invalidation strategy
   - **Mitigation**: Configurable cache duration

3. **Risk**: File format compatibility
   - **Mitigation**: Maintain existing LPC object format
   - **Mitigation**: Add format version checking
