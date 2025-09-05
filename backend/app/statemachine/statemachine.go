package statemachine

import (
	"sync"
)

// StateMachine defines a generic interface for state machines
type StateMachine[S any] interface {
	// CanTransition checks if a transition from one state to another is valid
	CanTransition(from S, to S) bool

	// Transition performs a state transition with validation
	Transition(from S, to S) error

	// GetTransitions returns all valid transitions from a given state
	GetTransitions(from S) []S
}

// TransitionRule defines a single state transition rule
type TransitionRule[S any] struct {
	From  S                      // Source state
	To    S                      // Target state
	Guard func(interface{}) bool // Optional validation function
}

// GenericStateMachine provides a generic implementation of StateMachine
type GenericStateMachine[S comparable] struct {
	transitions map[string]TransitionRule[S]
	mu          sync.RWMutex
}

// NewGenericStateMachine creates a new generic state machine
func NewGenericStateMachine[S comparable]() *GenericStateMachine[S] {
	return &GenericStateMachine[S]{
		transitions: make(map[string]TransitionRule[S]),
	}
}

// AddTransition adds a transition rule to the state machine
func (sm *GenericStateMachine[S]) AddTransition(from, to S, guard func(interface{}) bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.transitionKey(from, to)
	sm.transitions[key] = TransitionRule[S]{
		From:  from,
		To:    to,
		Guard: guard,
	}
}

// CanTransition checks if a transition is valid
func (sm *GenericStateMachine[S]) CanTransition(from S, to S) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := sm.transitionKey(from, to)
	rule, exists := sm.transitions[key]

	if !exists {
		return false
	}

	// If there's a guard, it must pass
	if rule.Guard != nil {
		// TODO: Guard validation will be implemented by specific state machines
		// This generic implementation cannot validate without context
		return true
	}

	return true
}

// Transition performs a state transition
func (sm *GenericStateMachine[S]) Transition(from S, to S) error {
	// TODO: Implement transition logic:
	// 1. Validate transition is allowed
	// 2. Execute any pre-transition hooks
	// 3. Perform the transition
	// 4. Execute any post-transition hooks
	// 5. Emit events if configured

	if !sm.CanTransition(from, to) {
		// TODO: Return proper error with details
		return nil
	}

	return nil
}

// GetTransitions returns all valid transitions from a state
func (sm *GenericStateMachine[S]) GetTransitions(from S) []S {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var validTransitions []S

	// TODO: Implement transition lookup:
	// 1. Find all transitions with matching 'from' state
	// 2. Check guards if present
	// 3. Return list of valid 'to' states

	for _, rule := range sm.transitions {
		// This is a simplified implementation - actual comparison depends on S type
		// TODO: Implement proper state comparison
		validTransitions = append(validTransitions, rule.To)
	}

	return validTransitions
}

// transitionKey generates a unique key for a transition
func (sm *GenericStateMachine[S]) transitionKey(from, to S) string {
	// TODO: Implement proper key generation based on state type
	// This is a placeholder - actual implementation depends on S type
	return ""
}

// GetAllTransitions returns all registered transition rules
func (sm *GenericStateMachine[S]) GetAllTransitions() []TransitionRule[S] {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var rules []TransitionRule[S]
	for _, rule := range sm.transitions {
		rules = append(rules, rule)
	}

	return rules
}

// RemoveTransition removes a transition rule
func (sm *GenericStateMachine[S]) RemoveTransition(from, to S) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.transitionKey(from, to)
	delete(sm.transitions, key)
}

// Clear removes all transition rules
func (sm *GenericStateMachine[S]) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.transitions = make(map[string]TransitionRule[S])
}
