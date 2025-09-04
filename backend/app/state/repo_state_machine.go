package state

import (
	"fmt"
	"sync"
	"time"
)

// StateTransition represents a single state change event
type StateTransition struct {
	RepoID    int        `json:"repoId"`
	From      RepoStatus `json:"from"`
	To        RepoStatus `json:"to"`
	Reason    string     `json:"reason"`
	Timestamp time.Time  `json:"timestamp"`
	Success   bool       `json:"success"`
	Error     string     `json:"error,omitempty"`
}

// TransitionHook defines callback functions for state transitions
type TransitionHook func(transition StateTransition) error

// RepoStateMachine manages repository state transitions with validation
type RepoStateMachine struct {
	mu            sync.RWMutex
	transitions   map[RepoStatus][]RepoStatus     // Valid transitions from each state
	preHooks      map[RepoStatus][]TransitionHook // Pre-transition hooks
	postHooks     map[RepoStatus][]TransitionHook // Post-transition hooks
	executor      *TransitionExecutor             // Reference to executor for transitions
	currentStates map[int]RepoStatus              // Track current state per repository
}

// NewRepoStateMachine creates a new repository state machine with predefined transitions
func NewRepoStateMachine() *RepoStateMachine {
	sm := &RepoStateMachine{
		transitions:   make(map[RepoStatus][]RepoStatus),
		preHooks:      make(map[RepoStatus][]TransitionHook),
		postHooks:     make(map[RepoStatus][]TransitionHook),
		currentStates: make(map[int]RepoStatus),
	}

	// Define valid state transitions
	sm.defineTransitions()
	return sm
}

// defineTransitions sets up the valid state transition matrix
func (sm *RepoStateMachine) defineTransitions() {
	// From Idle state
	sm.transitions[RepoStatusIdle] = []RepoStatus{
		RepoStatusBackingUp,
		RepoStatusPruning,
		RepoStatusDeleting,
		RepoStatusMounted,
		RepoStatusPerformingOperation,
		RepoStatusError, // TODO: I think this should not be possible
	}

	// From BackingUp state
	sm.transitions[RepoStatusBackingUp] = []RepoStatus{
		RepoStatusIdle,
		RepoStatusError,
	}

	// From Pruning state
	sm.transitions[RepoStatusPruning] = []RepoStatus{
		RepoStatusIdle,
		RepoStatusError,
	}

	// From Deleting state
	sm.transitions[RepoStatusDeleting] = []RepoStatus{
		RepoStatusIdle,
		RepoStatusError,
	}

	// From Mounted state
	sm.transitions[RepoStatusMounted] = []RepoStatus{
		RepoStatusIdle,
		RepoStatusError,
	}

	// From PerformingOperation state
	sm.transitions[RepoStatusPerformingOperation] = []RepoStatus{
		RepoStatusIdle,
		RepoStatusError,
	}

	// From Error state - can only go to Idle after resolution
	sm.transitions[RepoStatusError] = []RepoStatus{
		RepoStatusIdle,
	}
}

// CanTransition checks if a transition from one state to another is valid
func (sm *RepoStateMachine) CanTransition(from, to RepoStatus) (bool, string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	validTransitions, exists := sm.transitions[from]
	if !exists {
		return false, fmt.Sprintf("no transitions defined for state %s", from)
	}

	for _, validTo := range validTransitions {
		if validTo == to {
			return true, ""
		}
	}

	return false, fmt.Sprintf("invalid transition from %s to %s", from, to)
}

// ValidateTransition performs comprehensive validation for a state transition
func (sm *RepoStateMachine) ValidateTransition(repoID int, from, to RepoStatus, reason string) error {
	if reason == "" {
		return fmt.Errorf("transition reason is required")
	}

	if from == to {
		return fmt.Errorf("cannot transition to the same state (%s)", from)
	}

	canTransition, errMsg := sm.CanTransition(from, to)
	if !canTransition {
		return fmt.Errorf("transition validation failed for repo %d: %s", repoID, errMsg)
	}

	return nil
}

// ExecutePreHooks runs all pre-transition hooks for the target state
func (sm *RepoStateMachine) ExecutePreHooks(transition StateTransition) error {
	sm.mu.RLock()
	hooks := sm.preHooks[transition.To]
	sm.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(transition); err != nil {
			return fmt.Errorf("pre-transition hook failed: %w", err)
		}
	}
	return nil
}

// ExecutePostHooks runs all post-transition hooks for the target state
func (sm *RepoStateMachine) ExecutePostHooks(transition StateTransition) error {
	sm.mu.RLock()
	hooks := sm.postHooks[transition.To]
	sm.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(transition); err != nil {
			return fmt.Errorf("post-transition hook failed: %w", err)
		}
	}
	return nil
}

// AddPreHook adds a pre-transition hook for a specific state
func (sm *RepoStateMachine) AddPreHook(state RepoStatus, hook TransitionHook) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.preHooks[state] == nil {
		sm.preHooks[state] = make([]TransitionHook, 0)
	}
	sm.preHooks[state] = append(sm.preHooks[state], hook)
}

// AddPostHook adds a post-transition hook for a specific state
func (sm *RepoStateMachine) AddPostHook(state RepoStatus, hook TransitionHook) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.postHooks[state] == nil {
		sm.postHooks[state] = make([]TransitionHook, 0)
	}
	sm.postHooks[state] = append(sm.postHooks[state], hook)
}

// GetValidTransitions returns all valid transitions from a given state
func (sm *RepoStateMachine) GetValidTransitions(from RepoStatus) []RepoStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	validTransitions, exists := sm.transitions[from]
	if !exists {
		return []RepoStatus{}
	}

	// Return a copy to prevent external modification
	result := make([]RepoStatus, len(validTransitions))
	copy(result, validTransitions)
	return result
}

// GetCurrentState returns the current state of a repository
func (sm *RepoStateMachine) GetCurrentState(repoID int) RepoStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.currentStates[repoID]; exists {
		return state
	}
	return RepoStatusIdle // Default state
}

// SetCurrentState sets the current state of a repository
func (sm *RepoStateMachine) SetCurrentState(repoID int, state RepoStatus) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentStates[repoID] = state
}
