# Repository State Machine Design

## Package Structure
Location: `backend/app/statemachine/` (generic, reusable package)

## Core Types

### State Machine Interface
```go
type StateMachine[S any] interface {
    CanTransition(from S, to S) bool
    Transition(from S, to S) error
    GetTransitions(from S) []S
}
```

### Repository State Machine
```go
type RepositoryStateMachine struct {
    transitions map[string]TransitionRule
    mu          sync.RWMutex
}

type TransitionRule struct {
    From  RepositoryState
    To    RepositoryState
    Guard func(*Repository) bool // Optional validation
}
```

## Valid State Transitions

### From Idle
- Idle → Queued (operation added to queue)
- Idle → BackingUp (immediate backup start)
- Idle → Pruning (immediate prune start)
- Idle → Deleting (delete operation)
- Idle → Refreshing (refresh archives)
- Idle → Mounted (mount repository/archive)
- Idle → Error (unexpected error or locked repository)

### From Queued
- Queued → BackingUp (backup operation starts)
- Queued → Pruning (prune operation starts)
- Queued → Deleting (delete operation starts)
- Queued → Refreshing (refresh operation starts)
- Queued → Idle (queue cleared/expired)
- Queued → Error (queue processing error)

### From Active States (BackingUp, Pruning, Deleting, Refreshing)
- Active → Idle (operation completed)
- Active → Error (operation failed)
- Active → Queued (operation cancelled, queue not empty)

### From Mounted
- Mounted → Idle (unmounted)
- Mounted → Error (mount error)

### From Error
- Error → Idle (error cleared/resolved)

## Guard Conditions

```go
// Example guards
func canStartBackup(repo *Repository) bool {
    return repo.State != StateMounted && 
           repo.State != StateError
}

func canMount(repo *Repository) bool {
    return repo.State == StateIdle
}

func canQueue(repo *Repository) bool {
    return repo.State == StateIdle || 
           repo.State == StateQueued
}
```

## Implementation Methods

```go
// Check if transition is valid
func (sm *RepositoryStateMachine) CanTransition(repo *Repository, to RepositoryState) bool

// Perform state transition with validation
func (sm *RepositoryStateMachine) Transition(repo *Repository, to RepositoryState) error

// Get all valid transitions from current state
func (sm *RepositoryStateMachine) GetValidTransitions(repo *Repository) []RepositoryState

// Initialize with all transition rules
func NewRepositoryStateMachine() *RepositoryStateMachine
```

## Usage Example

```go
sm := statemachine.NewRepositoryStateMachine()

// Before queuing operation
if sm.CanTransition(repo, NewStateQueued(...)) {
    err := sm.Transition(repo, NewStateQueued(...))
    if err != nil {
        // Handle transition error
    }
}

// Check valid next states
validStates := sm.GetValidTransitions(repo)
```

## Thread Safety
- All state transitions must be protected by mutex
- State machine instance can be shared across goroutines
- Each repository maintains its own state

## Integration Points
1. **Queue Manager**: Uses state machine before processing operations
2. **Service Methods**: Validate transitions before operations
3. **Event System**: Emit events on state transitions
4. **Error Recovery**: Use state machine to determine recovery path