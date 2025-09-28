# Backend Development Guide

This guide covers backend-specific development patterns and tools for the Arco project.

## ADT (Algebraic Data Type) System

Arco uses a code generation system to create type-safe ADTs for state management and operation modeling.

### Overview

ADTs provide exhaustive pattern matching and type safety for complex state machines. The system uses the `adtenum` library with custom code generation to create:
- Type-safe variant wrappers
- Discriminated union types for Wails3 serialization
- Exhaustive type checking functions
- Constructor functions

### Defining ADTs

1. **Create variant structs** with data fields
2. **Define ADT type** using `adtenum.Enum[T]`
3. **Add marker methods** `isADTVariant()` for each variant

```go
// Variant structs
type Idle struct{}
type BackingUp struct {
    Progress *BackupProgress `json:"progress"`
}

// ADT definition
type RepositoryState adtenum.Enum[RepositoryState]

// Marker methods
func (Idle) isADTVariant() RepositoryState { var zero RepositoryState; return zero }
func (BackingUp) isADTVariant() RepositoryState { var zero RepositoryState; return zero }
```

### Generated Code

The generator creates in `adt.go`:
- **Discriminator enums**: `RepositoryStateType` with constants
- **Variant wrappers**: `IdleVariant`, `BackingUpVariant`
- **Constructors**: `NewRepositoryStateIdle()`, `NewRepositoryStateBackingUp()`
- **Union types**: `RepositoryStateUnion` for Wails3 serialization
- **Type functions**: `GetRepositoryStateType()` for exhaustive checking

### Usage

```go
// Create variants
state := NewRepositoryStateBackingUp(BackingUp{Progress: progress})

// Exhaustive switching
switch GetRepositoryStateType(state) {
case RepositoryStateTypeIdle:
    // Handle idle
case RepositoryStateTypeBackingUp:
    // Handle backing up
}

// Convert for frontend
union := ToRepositoryStateUnion(state)
```

### Code Generation

Run `go generate ./...` or `task dev:gen` to regenerate ADTs after changes.

The generator scans for `isADTVariant()` methods and creates consolidated `adt.go` files.