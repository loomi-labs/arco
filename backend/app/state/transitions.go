package state

import (
	"context"
	"fmt"
	"time"

	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/negrel/assert"
)

// TransitionContext provides context for state transitions
type TransitionContext struct {
	RepoID    int
	UserID    string
	RequestID string
	Context   context.Context
}

// TransitionResult represents the outcome of a state transition attempt
type TransitionResult struct {
	Success   bool          `json:"success"`
	From      RepoStatus    `json:"from"`
	To        RepoStatus    `json:"to"`
	Reason    string        `json:"reason"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// TransitionValidator defines a function that validates if a transition should be allowed
type TransitionValidator func(ctx TransitionContext, from, to RepoStatus) error

// BusinessRuleValidator encapsulates business logic for repository state transitions
type BusinessRuleValidator struct {
	validators map[string]TransitionValidator
}

// NewBusinessRuleValidator creates a validator with standard business rules
func NewBusinessRuleValidator() *BusinessRuleValidator {
	v := &BusinessRuleValidator{
		validators: make(map[string]TransitionValidator),
	}
	v.registerDefaultValidators()
	return v
}

// registerDefaultValidators sets up the standard business rule validators
func (v *BusinessRuleValidator) registerDefaultValidators() {
	// Prevent starting operations on busy repositories
	v.validators["no_concurrent_operations"] = func(ctx TransitionContext, from, to RepoStatus) error {
		busyStates := []RepoStatus{
			RepoStatusBackingUp,
			RepoStatusPruning,
			RepoStatusDeleting,
			RepoStatusPerformingOperation,
		}

		// If trying to transition to a busy state from another busy state
		if containsStatus(busyStates, from) && containsStatus(busyStates, to) && from != to {
			return fmt.Errorf("cannot start %s operation while repository is %s", to, from)
		}

		return nil
	}

	// Ensure mounted repositories are unmounted before operations
	v.validators["unmount_before_operations"] = func(ctx TransitionContext, from, to RepoStatus) error {
		dangerousOps := []RepoStatus{
			RepoStatusBackingUp,
			RepoStatusPruning,
			RepoStatusDeleting,
		}

		if from == RepoStatusMounted && containsStatus(dangerousOps, to) {
			return fmt.Errorf("repository must be unmounted before %s operation", to)
		}

		return nil
	}

	// Validate error state transitions
	v.validators["error_state_recovery"] = func(ctx TransitionContext, from, to RepoStatus) error {
		if from == RepoStatusError && to != RepoStatusIdle {
			return fmt.Errorf("repository in error state must be fixed before transitioning to %s", to)
		}

		return nil
	}
}

// ValidateTransition runs all registered validators against a transition
func (v *BusinessRuleValidator) ValidateTransition(ctx TransitionContext, from, to RepoStatus) error {
	for name, validator := range v.validators {
		if err := validator(ctx, from, to); err != nil {
			return fmt.Errorf("business rule '%s' failed: %w", name, err)
		}
	}
	return nil
}

// AddValidator registers a custom business rule validator
func (v *BusinessRuleValidator) AddValidator(name string, validator TransitionValidator) {
	v.validators[name] = validator
}

// RemoveValidator removes a business rule validator
func (v *BusinessRuleValidator) RemoveValidator(name string) {
	delete(v.validators, name)
}

// TransitionExecutor handles the actual execution of state transitions
type TransitionExecutor struct {
	stateMachine *RepoStateMachine
	validator    *BusinessRuleValidator
	eventEmitter types.EventEmitter
}

// NewTransitionExecutor creates a new transition executor
func NewTransitionExecutor(sm *RepoStateMachine, eventEmitter types.EventEmitter) *TransitionExecutor {
	assert.NotNil(eventEmitter, "eventEmitter must not be nil")
	return &TransitionExecutor{
		stateMachine: sm,
		validator:    NewBusinessRuleValidator(),
		eventEmitter: eventEmitter,
	}
}

// ExecuteTransition performs a complete state transition with validation, hooks, and audit
func (e *TransitionExecutor) ExecuteTransition(ctx TransitionContext, currentState, targetState RepoStatus) (*TransitionResult, error) {
	startTime := time.Now()

	result := &TransitionResult{
		From:      currentState,
		To:        targetState,
		Reason:    fmt.Sprintf("Transition from %s to %s", currentState, targetState),
		Timestamp: startTime,
	}

	// Step 1: Validate the transition
	if err := e.stateMachine.ValidateTransition(ctx.RepoID, currentState, targetState, ""); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("state machine validation failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Step 2: Run business rule validations
	if err := e.validator.ValidateTransition(ctx, currentState, targetState); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("business rule validation failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Step 3: Create transition record
	transition := StateTransition{
		RepoID:    ctx.RepoID,
		From:      currentState,
		To:        targetState,
		Reason:    fmt.Sprintf("Transition from %s to %s", currentState, targetState),
		Timestamp: startTime,
		Success:   false, // Will be updated on success
	}

	// Step 4: Execute pre-transition hooks
	if err := e.stateMachine.ExecutePreHooks(transition); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("pre-transition hook failed: %v", err)
		result.Duration = time.Since(startTime)
		transition.Success = false
		transition.Error = result.Error
		return result, err
	}

	// Step 5: Mark transition as successful
	transition.Success = true
	result.Success = true
	result.Duration = time.Since(startTime)

	// Step 6: Update current state
	e.stateMachine.SetCurrentState(ctx.RepoID, targetState)

	// Step 7: Execute post-transition hooks
	if err := e.stateMachine.ExecutePostHooks(transition); err != nil {
		// Post-hook failures are logged but don't fail the transition
		transition.Error = fmt.Sprintf("post-transition hook warning: %v", err)
	}

	// Step 8: Emit events
	e.eventEmitter.EmitEvent(ctx.Context, types.EventRepoStateChangedString(ctx.RepoID))

	return result, nil
}

// ForceTransition bypasses validation for emergency situations
func (e *TransitionExecutor) ForceTransition(ctx TransitionContext, currentState, targetState RepoStatus) *TransitionResult {
	startTime := time.Now()

	result := &TransitionResult{
		From:      currentState,
		To:        targetState,
		Reason:    fmt.Sprintf("FORCED: Transition from %s to %s", currentState, targetState),
		Timestamp: startTime,
		Success:   true,
	}

	transition := StateTransition{
		RepoID:    ctx.RepoID,
		From:      currentState,
		To:        targetState,
		Reason:    result.Reason,
		Timestamp: startTime,
		Success:   true,
	}

	// Update current state
	e.stateMachine.SetCurrentState(ctx.RepoID, targetState)

	// Still execute post-hooks if possible
	if err := e.stateMachine.ExecutePostHooks(transition); err != nil {
		transition.Error = fmt.Sprintf("post-transition hook warning: %v", err)
	}

	// Emit events
	e.eventEmitter.EmitEvent(ctx.Context, types.EventRepoStateChangedString(ctx.RepoID))

	result.Duration = time.Since(startTime)
	return result
}

// containsStatus checks if a slice contains a specific RepoStatus
func containsStatus(slice []RepoStatus, status RepoStatus) bool {
	for _, s := range slice {
		if s == status {
			return true
		}
	}
	return false
}

// GetTransitionReason returns a human-readable reason for common transitions
func GetTransitionReason(from, to RepoStatus, operation string) string {
	switch {
	case from == RepoStatusIdle && to == RepoStatusBackingUp:
		return fmt.Sprintf("Starting backup operation: %s", operation)
	case from == RepoStatusIdle && to == RepoStatusPruning:
		return fmt.Sprintf("Starting pruning operation: %s", operation)
	case from == RepoStatusIdle && to == RepoStatusDeleting:
		return fmt.Sprintf("Starting deletion operation: %s", operation)
	case from == RepoStatusIdle && to == RepoStatusMounted:
		return fmt.Sprintf("Mounting repository: %s", operation)
	case from == RepoStatusIdle && to == RepoStatusPerformingOperation:
		return fmt.Sprintf("Starting repository operation: %s", operation)
	case to == RepoStatusIdle:
		return fmt.Sprintf("Completed %s operation: %s", from, operation)
	case to == RepoStatusError:
		return fmt.Sprintf("Error during %s operation: %s", from, operation)
	default:
		return fmt.Sprintf("State transition %s -> %s: %s", from, to, operation)
	}
}
