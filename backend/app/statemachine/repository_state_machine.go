package statemachine

import (
	"fmt"
	"sync"
)

// RepositoryStateMachine manages state transitions for repositories
// This is a concrete implementation that will work with the repository ADT types
type RepositoryStateMachine struct {
	transitions map[string]TransitionRule[interface{}]
	mu          sync.RWMutex
}

// Repository interface for guard functions - will be implemented by repository.Repository
type Repository interface {
	GetState() interface{} // Returns RepositoryState ADT
	GetID() int
}

// TransitionRule for repository state machine with repository-specific guard
type RepositoryTransitionRule struct {
	From  string                // State type name (e.g., "Idle", "Queued")
	To    string                // State type name
	Guard func(Repository) bool // Repository-specific validation
}

// NewRepositoryStateMachine creates a new repository state machine with all valid transitions
func NewRepositoryStateMachine() *RepositoryStateMachine {
	sm := &RepositoryStateMachine{
		transitions: make(map[string]TransitionRule[interface{}]),
	}

	// Initialize all valid transitions based on STATEMACHINE_DESIGN.md
	sm.initializeTransitions()

	return sm
}

// CanTransition checks if a transition from one repository state to another is valid
func (sm *RepositoryStateMachine) CanTransition(repo Repository, toState interface{}) bool {
	// TODO: Implement repository state transition validation:
	// 1. Get current state from repository
	// 2. Determine state type names from ADT variants
	// 3. Look up transition rule
	// 4. Execute guard function if present
	// 5. Return validation result
	return false
}

// Transition performs a repository state transition with validation
func (sm *RepositoryStateMachine) Transition(repo Repository, toState interface{}) error {
	// TODO: Implement repository state transition:
	// 1. Validate transition is allowed via CanTransition
	// 2. Perform any pre-transition actions
	// 3. Update repository state (will be done by service layer)
	// 4. Execute any post-transition actions
	// 5. Emit state change events (will be done by service layer)

	if !sm.CanTransition(repo, toState) {
		return fmt.Errorf("invalid state transition for repository %d", repo.GetID())
	}

	return nil
}

// GetValidTransitions returns all valid transitions from the repository's current state
func (sm *RepositoryStateMachine) GetValidTransitions(repo Repository) []string {
	// TODO: Implement valid transition lookup:
	// 1. Get current state from repository
	// 2. Find all transitions with matching 'from' state
	// 3. Check guards for each transition
	// 4. Return list of valid target state names
	return nil
}

// initializeTransitions sets up all valid state transitions
func (sm *RepositoryStateMachine) initializeTransitions() {
	// TODO: Initialize all transitions based on STATEMACHINE_DESIGN.md

	// From Idle
	sm.addTransition("Idle", "Queued", sm.canQueue)
	sm.addTransition("Idle", "BackingUp", sm.canStartBackup)
	sm.addTransition("Idle", "Pruning", sm.canStartBackup)
	sm.addTransition("Idle", "Deleting", sm.canStartOperation)
	sm.addTransition("Idle", "Refreshing", sm.canStartOperation)
	sm.addTransition("Idle", "Mounted", sm.canMount)
	sm.addTransition("Idle", "Error", nil) // Always allowed for unexpected errors

	// From Queued
	sm.addTransition("Queued", "BackingUp", sm.canStartBackup)
	sm.addTransition("Queued", "Pruning", sm.canStartBackup)
	sm.addTransition("Queued", "Deleting", sm.canStartOperation)
	sm.addTransition("Queued", "Refreshing", sm.canStartOperation)
	sm.addTransition("Queued", "Idle", nil)  // Queue cleared/expired
	sm.addTransition("Queued", "Error", nil) // Queue processing error

	// From Active States (BackingUp, Pruning, Deleting, Refreshing)
	activeStates := []string{"BackingUp", "Pruning", "Deleting", "Refreshing"}
	for _, state := range activeStates {
		sm.addTransition(state, "Idle", nil)                      // Operation completed
		sm.addTransition(state, "Error", nil)                     // Operation failed
		sm.addTransition(state, "Queued", sm.hasQueuedOperations) // Operation cancelled, queue not empty
	}

	// From Mounted
	sm.addTransition("Mounted", "Idle", nil)  // Unmounted
	sm.addTransition("Mounted", "Error", nil) // Mount error

	// From Error
	sm.addTransition("Error", "Idle", nil) // Error cleared/resolved
}

// addTransition adds a transition rule with optional guard
func (sm *RepositoryStateMachine) addTransition(from, to string, guard func(Repository) bool) {
	key := sm.transitionKey(from, to)

	// Convert guard to interface{} compatible function
	var genericGuard func(interface{}) bool
	if guard != nil {
		genericGuard = func(entity interface{}) bool {
			if repo, ok := entity.(Repository); ok {
				return guard(repo)
			}
			return false
		}
	}

	sm.transitions[key] = TransitionRule[interface{}]{
		From:  from,
		To:    to,
		Guard: genericGuard,
	}
}

// transitionKey generates a unique key for a state transition
func (sm *RepositoryStateMachine) transitionKey(from, to string) string {
	return fmt.Sprintf("%s->%s", from, to)
}

// ============================================================================
// GUARD CONDITIONS
// ============================================================================

// canStartBackup checks if a backup operation can start
func (sm *RepositoryStateMachine) canStartBackup(repo Repository) bool {
	// TODO: Implement guard condition:
	// 1. Get current state from repository
	// 2. Check state is not Mounted and not Error
	// 3. Check repository is accessible
	return false
}

// canMount checks if repository can be mounted
func (sm *RepositoryStateMachine) canMount(repo Repository) bool {
	// TODO: Implement guard condition:
	// 1. Get current state from repository
	// 2. Check state is Idle
	// 3. Check repository is not already mounted
	return false
}

// canQueue checks if operations can be queued
func (sm *RepositoryStateMachine) canQueue(repo Repository) bool {
	// TODO: Implement guard condition:
	// 1. Get current state from repository
	// 2. Check state is Idle or Queued
	// 3. Check repository is accessible
	return false
}

// canStartOperation checks if a general operation can start
func (sm *RepositoryStateMachine) canStartOperation(repo Repository) bool {
	// TODO: Implement guard condition:
	// 1. Get current state from repository
	// 2. Check state allows new operations
	// 3. Check repository is accessible
	return false
}

// hasQueuedOperations checks if repository has queued operations
func (sm *RepositoryStateMachine) hasQueuedOperations(repo Repository) bool {
	// TODO: Implement guard condition:
	// 1. Check if repository has operations in queue
	// 2. This will require access to QueueManager
	return false
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// getStateTypeName extracts the state type name from an ADT variant
func (sm *RepositoryStateMachine) getStateTypeName(state interface{}) string {
	// TODO: Implement state type extraction from ADT:
	// 1. Use type assertion or reflection to determine variant type
	// 2. Return corresponding state name (e.g., "Idle", "Queued", etc.)
	// 3. Handle all repository state variants
	return "Unknown"
}

// validateStateTransition performs comprehensive validation
func (sm *RepositoryStateMachine) validateStateTransition(repo Repository, from, to string) error {
	// TODO: Implement comprehensive validation:
	// 1. Check if transition rule exists
	// 2. Execute guard function if present
	// 3. Perform any additional business logic validation
	// 4. Return detailed error if validation fails
	return nil
}

// GetTransitionRules returns all registered transition rules for inspection
func (sm *RepositoryStateMachine) GetTransitionRules() []RepositoryTransitionRule {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var rules []RepositoryTransitionRule
	for _, rule := range sm.transitions {
		// TODO: Convert internal rules to external format
		rules = append(rules, RepositoryTransitionRule{
			From: rule.From.(string),
			To:   rule.To.(string),
			// Guard function conversion would be complex, leave nil for now
		})
	}

	return rules
}
