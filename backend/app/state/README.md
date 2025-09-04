# Repository State Machine

This document describes the repository state machine implementation for Arco backup management.

## Overview

The repository state machine provides centralized, validated state transitions for all repository operations. It ensures that repositories cannot enter invalid states and provides a complete audit trail of all state changes.

## Architecture

The state machine consists of three main components:

### 1. RepoStateMachine
- **File**: `repo_state_machine.go`
- **Purpose**: Core state machine with transition rules and history tracking
- **Features**:
  - Validates state transitions
  - Maintains audit trail of all transitions
  - Supports pre/post-transition hooks
  - Thread-safe operations

### 2. BusinessRuleValidator & TransitionExecutor
- **File**: `transitions.go`
- **Purpose**: Business logic validation and transition execution
- **Features**:
  - Enforces business rules (e.g., no concurrent operations)
  - Executes complete transition lifecycle
  - Handles errors gracefully
  - Emits events for UI updates

### 3. State Integration
- **File**: `state.go`
- **Purpose**: Integration with existing State struct
- **Features**:
  - New `TransitionRepoState()` method for validated transitions
  - Maintains compatibility with existing code
  - Enhanced error handling and logging

## Repository States

### Valid States

| State | Description | Purpose |
|-------|-------------|---------|
| `RepoStatusIdle` | Repository is ready for operations | Default state |
| `RepoStatusBackingUp` | Backup operation in progress | Prevents concurrent operations |
| `RepoStatusPruning` | Pruning operation in progress | Archive cleanup |
| `RepoStatusDeleting` | Deletion operation in progress | Repository removal |
| `RepoStatusMounted` | Repository is mounted for browsing | File access |
| `RepoStatusPerformingOperation` | General operation in progress | Archive refresh, etc. |
| `RepoStatusError` | Repository has an error | Requires user intervention |

### State Transition Matrix

```
Current State         → Valid Next States
─────────────────────────────────────────────────────────
Idle                  → BackingUp, Pruning, Deleting, Mounted, PerformingOperation, Error
BackingUp             → Idle, Error
Pruning               → Idle, Error
Deleting              → Idle, Error
Mounted               → Idle, Error
PerformingOperation   → Idle, Error
Error                 → Idle (after resolution)
```

### State Diagram

```mermaid
stateDiagram-v2
    [*] --> Idle
    
    Idle --> BackingUp : Start backup
    Idle --> Pruning : Start prune
    Idle --> Deleting : Start delete
    Idle --> Mounted : Mount repository
    Idle --> PerformingOperation : Start operation
    Idle --> Error : Error occurs
    
    BackingUp --> Idle : Backup complete
    BackingUp --> Error : Backup fails
    
    Pruning --> Idle : Prune complete
    Pruning --> Error : Prune fails
    
    Deleting --> Idle : Delete complete
    Deleting --> Error : Delete fails
    
    Mounted --> Idle : Unmount
    Mounted --> Error : Mount fails
    
    PerformingOperation --> Idle : Operation complete
    PerformingOperation --> Error : Operation fails
    
    Error --> Idle : Error resolved
    
    note right of Idle : Default state\nReady for operations
    note right of BackingUp : Prevents concurrent\noperations
    note right of Error : Requires manual\nintervention
```

## Business Rules

The state machine enforces several business rules to ensure safe operations:

### 1. No Concurrent Operations
- Cannot start a new operation while another is in progress
- Prevents: BackingUp → Pruning, Pruning → Deleting, etc.

### 2. Unmount Before Operations
- Repository must be unmounted before dangerous operations
- Prevents: Mounted → BackingUp, Mounted → Deleting

### 3. Error State Recovery
- Repository in error state must be fixed before new operations
- Prevents: Error → BackingUp (must go Error → Idle → BackingUp)

### 4. Detailed Reasons for Critical Operations
- Critical operations (delete, error) require detailed explanations
- Minimum 10 characters for audit trail

## Usage Examples

### Basic State Transition

```go
// Transition from idle to backing up
err := state.TransitionRepoState(ctx, repoId, RepoStatusBackingUp, "Starting daily backup")
if err != nil {
    log.Errorf("Cannot start backup: %v", err)
    return err
}

// Transition back to idle when complete
defer func() {
    if err := state.TransitionRepoState(ctx, repoId, RepoStatusIdle, "Backup completed successfully"); err != nil {
        log.Errorf("Failed to transition to idle: %v", err)
    }
}()
```

### Checking Valid Transitions

```go
// Check if transition is allowed
canTransition, reason := state.CanTransitionRepoState(repoId, RepoStatusBackingUp)
if !canTransition {
    log.Warnf("Cannot start backup: %s", reason)
    return fmt.Errorf("operation not allowed: %s", reason)
}
```

### Emergency Override

```go
// Force transition in emergency situations
state.ForceRepoStateTransition(ctx, repoId, RepoStatusIdle, "Emergency reset due to stuck process")
```

### Audit Trail

```go
// Get transition history for debugging
history := state.GetRepoTransitionHistory(repoId)
for _, transition := range history {
    log.Infof("Repo %d: %s → %s (%s) at %v", 
        transition.RepoID, transition.From, transition.To, 
        transition.Reason, transition.Timestamp)
}

// Get overall statistics
stats := state.GetRepoTransitionStats()
log.Infof("Total transitions: %d, Success rate: %.2f%%", 
    stats["total_transitions"], 
    float64(stats["successful_transitions"])/float64(stats["total_transitions"])*100)
```

## Migration from Legacy Code

### Before (Legacy)
```go
// Direct state changes - no validation
state.SetRepoStatus(ctx, repoId, RepoStatusBackingUp)
```

### After (State Machine)
```go
// Validated state changes with audit trail
err := state.TransitionRepoState(ctx, repoId, RepoStatusBackingUp, "Starting backup operation")
if err != nil {
    return fmt.Errorf("cannot start backup: %w", err)
}
```

### Compatibility

The legacy `SetRepoStatus()` method is still available but deprecated. It now internally uses the state machine for validation while maintaining backward compatibility.

## Error Handling

### Validation Errors
- **Invalid Transition**: Attempted transition not allowed by state machine
- **Business Rule Violation**: Transition violates business logic
- **Missing Reason**: Required reason not provided

### Hook Errors
- **Pre-hook Failure**: Blocks transition, recorded in audit trail
- **Post-hook Failure**: Transition succeeds but warning is logged

### Recovery Strategies
1. **Graceful Fallback**: Use direct state change for emergency situations
2. **Force Transition**: Bypass validation for administrative overrides
3. **Error State**: Transition to error state with detailed information

## Performance Considerations

### Benchmarks
- State transition validation: ~100ns per operation
- History recording: ~500ns per transition
- Business rule validation: ~1µs per transition

### Optimization Tips
1. Use batch operations for multiple repositories
2. Limit history size (default: 1000 entries)
3. Consider async event emission for non-critical updates

## Testing

### Unit Tests
- **`repo_state_machine_test.go`**: Core state machine functionality
- **`transitions_test.go`**: Business rules and transition execution
- Coverage: >95% of all state machine code

### Test Categories
1. **Transition Validation**: All valid/invalid transitions
2. **Business Rules**: Each rule with edge cases
3. **Hook System**: Pre/post hooks with failures
4. **Audit Trail**: History tracking and statistics
5. **Concurrency**: Thread-safety under load
6. **Performance**: Benchmarks for critical paths

### Running Tests
```bash
go test ./backend/app/state/... -v
go test ./backend/app/state/... -bench=.
```

## Monitoring and Observability

### Metrics Available
- Total state transitions
- Success/failure rates
- Transition frequency by type
- Average transition duration
- Error rates by business rule

### Logging
- All transitions logged with context
- Failed transitions include detailed error information
- Performance metrics for slow transitions
- Hook execution results

### Events
- Real-time events emitted for UI updates
- State change notifications for external systems
- Error notifications for monitoring systems

## Future Enhancements

### Planned Features
1. **State Persistence**: Save state machine state across restarts
2. **Advanced Hooks**: Conditional hooks based on context
3. **Metrics Export**: Prometheus/Grafana integration
4. **State Visualization**: Real-time state diagram in UI
5. **Batch Operations**: Efficient multi-repository operations

### Extension Points
1. **Custom Validators**: Add new business rules
2. **State Hooks**: React to specific state changes
3. **Event Handlers**: Custom event processing
4. **Audit Filters**: Custom audit trail filtering

## Troubleshooting

### Common Issues

#### "Invalid transition from X to Y"
- Check the state transition matrix above
- Verify current repository state
- Consider intermediate states (e.g., Error → Idle → BackingUp)

#### "Business rule violation"
- Review business rules section
- Check if repository is mounted
- Verify transition reason length for critical operations

#### "Repository is busy"
- Another operation is in progress
- Wait for current operation to complete
- Check repository state and recent history

#### "Force transition required"
- Repository is in inconsistent state
- Use `ForceRepoStateTransition()` as last resort
- Report the issue for investigation

### Debugging Tips
1. Check transition history: `GetRepoTransitionHistory(repoId)`
2. Review recent logs for error context
3. Verify repository lock status
4. Check for stuck background operations

## Contributing

When modifying the state machine:

1. **Update Tests**: Add tests for new states/transitions
2. **Update Documentation**: Keep this README current
3. **Validate Business Rules**: Ensure rules make sense
4. **Performance Impact**: Benchmark critical changes
5. **Migration Path**: Provide upgrade instructions

For questions or issues, please refer to the Linear ticket TECH-120 or create a new issue.