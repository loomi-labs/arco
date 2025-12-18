package statemachine

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/negrel/assert"
)

// RepositoryStateMachine manages state transitions for repositories
// This is a concrete implementation that will work with the repository ADT types
type RepositoryStateMachine struct {
	transitions  map[transitionKey]RepositoryTransitionRule
	mu           sync.RWMutex
	queueManager QueueManager
}

type QueueManager interface {
	// HasQueuedOperations checks if a repository has pending operations
	HasQueuedOperations(repoID int) bool
}

// transitionKey uniquely identifies a state transition using type information
type transitionKey struct {
	fromType reflect.Type
	toType   reflect.Type
}

// RepositoryTransitionRule defines a repository state machine transition with optional guard
type RepositoryTransitionRule struct {
	From  RepositoryState                              // Source state
	To    RepositoryState                              // Target state
	Guard func(repoId int, state RepositoryState) bool // Repository-specific validation
}

// NewRepositoryStateMachine creates a new repository state machine with all valid transitions
func NewRepositoryStateMachine() *RepositoryStateMachine {
	sm := &RepositoryStateMachine{
		transitions: make(map[transitionKey]RepositoryTransitionRule),
	}

	// Initialize all valid transitions
	sm.initializeTransitions()

	return sm
}

func (sm *RepositoryStateMachine) SetQueueManager(queueManager QueueManager) {
	sm.queueManager = queueManager
}

// CanTransition checks if a transition from one repository state to another is valid
func (sm *RepositoryStateMachine) CanTransition(repoId int, currentState RepositoryState, toState RepositoryState) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := sm.createTransitionKey(currentState, toState)

	rule, exists := sm.transitions[key]
	if !exists {
		return false
	}

	// Execute guard
	return rule.Guard(repoId, currentState)
}

// Transition performs a repository state transition with validation
func (sm *RepositoryStateMachine) Transition(repoId int, currentState RepositoryState, toState RepositoryState) error {
	if !sm.CanTransition(repoId, currentState, toState) {
		return fmt.Errorf("invalid state transition for repository %d: %s -> %s",
			repoId,
			GetStateTypeName(currentState),
			GetStateTypeName(toState))
	}

	// Transition is valid - actual state update will be done by service layer
	// This method validates and can perform pre/post transition hooks
	return nil
}

// GetValidTransitions returns all valid transitions from the repository's current state
func (sm *RepositoryStateMachine) GetValidTransitions(repoId int, currentState RepositoryState) []RepositoryState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	currentType := reflect.TypeOf(currentState)

	var validStates []RepositoryState
	for key, rule := range sm.transitions {
		if key.fromType == currentType {
			// Check guard if present
			if rule.Guard(repoId, currentState) {
				validStates = append(validStates, rule.To)
			}
		}
	}

	return validStates
}

// TransitionDef defines a single transition with ADT types
type TransitionDef struct {
	From  RepositoryState
	To    RepositoryState
	Guard func(repoId int, currentState RepositoryState) bool
}

// initializeTransitions sets up all valid state transitions using a map-based approach
func (sm *RepositoryStateMachine) initializeTransitions() {
	// Create state instances once
	idle := NewRepositoryStateIdle(Idle{})
	queued := NewRepositoryStateQueued(Queued{})
	backingUp := NewRepositoryStateBackingUp(BackingUp{})
	pruning := NewRepositoryStatePruning(Pruning{})
	deleting := NewRepositoryStateDeleting(Deleting{})
	refreshing := NewRepositoryStateRefreshing(Refreshing{})
	checking := NewRepositoryStateChecking(Checking{})
	mounting := NewRepositoryStateMounting(Mounting{})
	mounted := NewRepositoryStateMounted(Mounted{})
	errorState := NewRepositoryStateError(Error{})

	// No guard needed for this transition
	nop := func(repoId int, currentState RepositoryState) bool {
		return true
	}

	// Repository can not have a queued operation
	hasQueuedOps := func(repoId int, state RepositoryState) bool {
		assert.NotNil(sm.queueManager, "queueManager must be initialized before use - this is a programming error")
		return sm.queueManager.HasQueuedOperations(repoId)
	}

	// Define all transitions using ADT types directly
	transitions := []TransitionDef{
		// From Idle
		{From: idle, To: queued, Guard: nop},     // New operation added to queue when repo is idle
		{From: idle, To: backingUp, Guard: nop},  // Start backup immediately (no queue)
		{From: idle, To: pruning, Guard: nop},    // Start prune immediately (no queue)
		{From: idle, To: deleting, Guard: nop},   // Start repository delete operation
		{From: idle, To: refreshing, Guard: nop}, // Start refreshing archive list
		{From: idle, To: checking, Guard: nop},   // Start checking repository integrity
		{From: idle, To: mounting, Guard: nop},   // Start mounting repository or archive
		{From: idle, To: errorState, Guard: nop}, // Unexpected error (e.g., repository locked)

		// From Queued
		{From: queued, To: backingUp, Guard: nop},  // Backup operation starts from queue
		{From: queued, To: pruning, Guard: nop},    // Prune operation starts from queue
		{From: queued, To: deleting, Guard: nop},   // Delete operation starts from queue
		{From: queued, To: refreshing, Guard: nop}, // Refresh operation starts from queue
		{From: queued, To: checking, Guard: nop},   // Check operation starts from queue
		{From: queued, To: mounting, Guard: nop},   // Mount operation starts from queue
		{From: queued, To: idle, Guard: nop},       // Queue cleared or all operations expired
		{From: queued, To: errorState, Guard: nop}, // Queue processing error

		// From BackingUp
		{From: backingUp, To: idle, Guard: nop},            // Backup completed successfully
		{From: backingUp, To: errorState, Guard: nop},      // Backup failed with error
		{From: backingUp, To: queued, Guard: hasQueuedOps}, // Backup cancelled, more operations waiting

		// From Pruning
		{From: pruning, To: idle, Guard: nop},            // Prune completed successfully
		{From: pruning, To: errorState, Guard: nop},      // Prune failed with error
		{From: pruning, To: queued, Guard: hasQueuedOps}, // Prune cancelled, more operations waiting

		// From Deleting
		{From: deleting, To: idle, Guard: nop},            // Delete completed successfully
		{From: deleting, To: errorState, Guard: nop},      // Delete failed with error
		{From: deleting, To: queued, Guard: hasQueuedOps}, // Delete cancelled, more operations waiting

		// From Refreshing
		{From: refreshing, To: idle, Guard: nop},            // Refresh completed successfully
		{From: refreshing, To: errorState, Guard: nop},      // Refresh failed with error
		{From: refreshing, To: queued, Guard: hasQueuedOps}, // Refresh cancelled, more operations waiting

		// From Checking
		{From: checking, To: idle, Guard: nop},            // Check completed successfully
		{From: checking, To: errorState, Guard: nop},      // Check failed with error
		{From: checking, To: queued, Guard: hasQueuedOps}, // Check cancelled, more operations waiting

		// From Mounting
		{From: mounting, To: mounted, Guard: nop},         // Mount completed successfully
		{From: mounting, To: errorState, Guard: nop},      // Mount failed with error
		{From: mounting, To: queued, Guard: hasQueuedOps}, // Mount cancelled, more operations waiting

		// From Mounted
		{From: mounted, To: refreshing, Guard: nop}, // Repository/archive unmounting
		{From: mounted, To: errorState, Guard: nop}, // Mount error (e.g., filesystem issue)

		// From Error
		{From: errorState, To: idle, Guard: nop}, // Error resolved/cleared by user action
	}

	// Build the transitions map
	for _, def := range transitions {
		key := transitionKey{
			fromType: reflect.TypeOf(def.From),
			toType:   reflect.TypeOf(def.To),
		}

		sm.transitions[key] = RepositoryTransitionRule{
			From:  def.From,
			To:    def.To,
			Guard: def.Guard,
		}
	}
}

// createTransitionKey generates a transition key from state instances
func (sm *RepositoryStateMachine) createTransitionKey(from, to RepositoryState) transitionKey {
	return transitionKey{
		fromType: reflect.TypeOf(from),
		toType:   reflect.TypeOf(to),
	}
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// CanTransitionToAny checks if any of the given states are valid transitions
func (sm *RepositoryStateMachine) CanTransitionToAny(repoId int, currentState RepositoryState, states ...RepositoryState) bool {
	for _, state := range states {
		if sm.CanTransition(repoId, currentState, state) {
			return true
		}
	}
	return false
}

// GetAllPossibleStates returns all possible states that can be transitioned to from any state
func (sm *RepositoryStateMachine) GetAllPossibleStates() []RepositoryState {
	return []RepositoryState{
		NewRepositoryStateIdle(Idle{}),
		NewRepositoryStateQueued(Queued{}),
		NewRepositoryStateBackingUp(BackingUp{}),
		NewRepositoryStatePruning(Pruning{}),
		NewRepositoryStateDeleting(Deleting{}),
		NewRepositoryStateRefreshing(Refreshing{}),
		NewRepositoryStateMounted(Mounted{}),
		NewRepositoryStateError(Error{}),
	}
}

// GetStateTypeName is now provided by the states.go file
// This method is kept for backward compatibility
func (sm *RepositoryStateMachine) getStateTypeName(state RepositoryState) string {
	return GetStateTypeName(state)
}

// validateStateTransition performs comprehensive validation
func (sm *RepositoryStateMachine) validateStateTransition(repoId int, currentState RepositoryState, to RepositoryState) error {
	key := sm.createTransitionKey(currentState, to)

	sm.mu.RLock()
	rule, exists := sm.transitions[key]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no transition rule found for %s -> %s",
			GetStateTypeName(currentState),
			GetStateTypeName(to))
	}

	// Execute guard function
	if !rule.Guard(repoId, currentState) {
		return fmt.Errorf("guard condition failed for transition %s -> %s",
			GetStateTypeName(currentState),
			GetStateTypeName(to))
	}

	return nil
}
