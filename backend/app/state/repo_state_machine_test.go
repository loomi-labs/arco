package state

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRepoStateMachine(t *testing.T) {
	sm := NewRepoStateMachine()
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.transitions)
	assert.Equal(t, 1000, sm.maxHistory)
}

func TestRepoStateMachine_ValidTransitions(t *testing.T) {
	sm := NewRepoStateMachine()

	tests := []struct {
		name     string
		from     RepoStatus
		to       RepoStatus
		expected bool
	}{
		// Valid transitions from Idle
		{"Idle to BackingUp", RepoStatusIdle, RepoStatusBackingUp, true},
		{"Idle to Pruning", RepoStatusIdle, RepoStatusPruning, true},
		{"Idle to Deleting", RepoStatusIdle, RepoStatusDeleting, true},
		{"Idle to Mounted", RepoStatusIdle, RepoStatusMounted, true},
		{"Idle to PerformingOperation", RepoStatusIdle, RepoStatusPerformingOperation, true},
		{"Idle to Error", RepoStatusIdle, RepoStatusError, true},

		// Valid transitions from BackingUp
		{"BackingUp to Idle", RepoStatusBackingUp, RepoStatusIdle, true},
		{"BackingUp to Error", RepoStatusBackingUp, RepoStatusError, true},

		// Valid transitions from Pruning
		{"Pruning to Idle", RepoStatusPruning, RepoStatusIdle, true},
		{"Pruning to Error", RepoStatusPruning, RepoStatusError, true},

		// Valid transitions from Deleting
		{"Deleting to Idle", RepoStatusDeleting, RepoStatusIdle, true},
		{"Deleting to Error", RepoStatusDeleting, RepoStatusError, true},

		// Valid transitions from Mounted
		{"Mounted to Idle", RepoStatusMounted, RepoStatusIdle, true},
		{"Mounted to Error", RepoStatusMounted, RepoStatusError, true},

		// Valid transitions from PerformingOperation
		{"PerformingOperation to Idle", RepoStatusPerformingOperation, RepoStatusIdle, true},
		{"PerformingOperation to Error", RepoStatusPerformingOperation, RepoStatusError, true},

		// Valid transitions from Error
		{"Error to Idle", RepoStatusError, RepoStatusIdle, true},

		// Invalid transitions
		{"BackingUp to Pruning", RepoStatusBackingUp, RepoStatusPruning, false},
		{"BackingUp to Deleting", RepoStatusBackingUp, RepoStatusDeleting, false},
		{"BackingUp to Mounted", RepoStatusBackingUp, RepoStatusMounted, false},
		{"BackingUp to PerformingOperation", RepoStatusBackingUp, RepoStatusPerformingOperation, false},
		{"Pruning to BackingUp", RepoStatusPruning, RepoStatusBackingUp, false},
		{"Pruning to Deleting", RepoStatusPruning, RepoStatusDeleting, false},
		{"Pruning to Mounted", RepoStatusPruning, RepoStatusMounted, false},
		{"Deleting to BackingUp", RepoStatusDeleting, RepoStatusBackingUp, false},
		{"Deleting to Pruning", RepoStatusDeleting, RepoStatusPruning, false},
		{"Deleting to Mounted", RepoStatusDeleting, RepoStatusMounted, false},
		{"Error to BackingUp", RepoStatusError, RepoStatusBackingUp, false},
		{"Error to Pruning", RepoStatusError, RepoStatusPruning, false},
		{"Error to Deleting", RepoStatusError, RepoStatusDeleting, false},
		{"Error to Mounted", RepoStatusError, RepoStatusMounted, false},
		{"Error to PerformingOperation", RepoStatusError, RepoStatusPerformingOperation, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canTransition, reason := sm.CanTransition(tt.from, tt.to)
			assert.Equal(t, tt.expected, canTransition, "Unexpected transition result: %s", reason)

			if tt.expected {
				assert.Empty(t, reason, "Valid transition should have empty reason")
			} else {
				assert.NotEmpty(t, reason, "Invalid transition should have a reason")
			}
		})
	}
}

func TestRepoStateMachine_ValidateTransition(t *testing.T) {
	sm := NewRepoStateMachine()

	// Test valid transition
	err := sm.ValidateTransition(1, RepoStatusIdle, RepoStatusBackingUp, "Starting backup")
	assert.NoError(t, err)

	// Test invalid transition
	err = sm.ValidateTransition(1, RepoStatusBackingUp, RepoStatusPruning, "Invalid transition")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition from")

	// Test empty reason
	err = sm.ValidateTransition(1, RepoStatusIdle, RepoStatusBackingUp, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transition reason is required")

	// Test same state transition
	err = sm.ValidateTransition(1, RepoStatusIdle, RepoStatusIdle, "Same state")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot transition to the same state")
}

func TestRepoStateMachine_Hooks(t *testing.T) {
	sm := NewRepoStateMachine()

	var preHookCalled bool
	var postHookCalled bool

	preHook := func(transition StateTransition) error {
		preHookCalled = true
		assert.Equal(t, RepoStatusBackingUp, transition.To)
		return nil
	}

	postHook := func(transition StateTransition) error {
		postHookCalled = true
		assert.Equal(t, RepoStatusBackingUp, transition.To)
		return nil
	}

	sm.AddPreHook(RepoStatusBackingUp, preHook)
	sm.AddPostHook(RepoStatusBackingUp, postHook)

	transition := StateTransition{
		RepoID: 1,
		From:   RepoStatusIdle,
		To:     RepoStatusBackingUp,
		Reason: "Test hooks",
	}

	// Execute hooks
	err := sm.ExecutePreHooks(transition)
	assert.NoError(t, err)
	assert.True(t, preHookCalled)

	err = sm.ExecutePostHooks(transition)
	assert.NoError(t, err)
	assert.True(t, postHookCalled)
}

func TestRepoStateMachine_HookError(t *testing.T) {
	sm := NewRepoStateMachine()

	failingHook := func(transition StateTransition) error {
		return fmt.Errorf("hook failed")
	}

	sm.AddPreHook(RepoStatusBackingUp, failingHook)

	transition := StateTransition{
		RepoID: 1,
		From:   RepoStatusIdle,
		To:     RepoStatusBackingUp,
		Reason: "Test hook failure",
	}

	err := sm.ExecutePreHooks(transition)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pre-transition hook failed")
}

func TestRepoStateMachine_GetValidTransitions(t *testing.T) {
	sm := NewRepoStateMachine()

	// Test idle state transitions
	idleTransitions := sm.GetValidTransitions(RepoStatusIdle)
	expectedIdle := []RepoStatus{
		RepoStatusBackingUp,
		RepoStatusPruning,
		RepoStatusDeleting,
		RepoStatusMounted,
		RepoStatusPerformingOperation,
		RepoStatusError,
	}
	assert.ElementsMatch(t, expectedIdle, idleTransitions)

	// Test backing up state transitions
	backingUpTransitions := sm.GetValidTransitions(RepoStatusBackingUp)
	expectedBackingUp := []RepoStatus{RepoStatusIdle, RepoStatusError}
	assert.ElementsMatch(t, expectedBackingUp, backingUpTransitions)

	// Test error state transitions
	errorTransitions := sm.GetValidTransitions(RepoStatusError)
	expectedError := []RepoStatus{RepoStatusIdle}
	assert.ElementsMatch(t, expectedError, errorTransitions)

	// Test unknown state
	unknownTransitions := sm.GetValidTransitions(RepoStatus("unknown"))
	assert.Empty(t, unknownTransitions)
}

func TestRepoStateMachine_GetTransitionStats(t *testing.T) {
	sm := NewRepoStateMachine()

	// Record some transitions
	transitions := []StateTransition{
		{RepoID: 1, From: RepoStatusIdle, To: RepoStatusBackingUp, Success: true},
		{RepoID: 1, From: RepoStatusBackingUp, To: RepoStatusIdle, Success: true},
		{RepoID: 1, From: RepoStatusIdle, To: RepoStatusBackingUp, Success: false, Error: "test error"},
		{RepoID: 2, From: RepoStatusIdle, To: RepoStatusPruning, Success: true},
	}

	for _, transition := range transitions {
		sm.RecordTransition(transition)
	}

	stats := sm.GetTransitionStats()

	assert.Equal(t, 4, stats["total_transitions"])
	assert.Equal(t, 3, stats["successful_transitions"])
	assert.Equal(t, 1, stats["failed_transitions"])
	assert.Equal(t, 2, stats["idle->backingUp"])
	assert.Equal(t, 1, stats["backingUp->idle"])
	assert.Equal(t, 1, stats["idle->pruning"])
}

// Benchmark tests
func BenchmarkRepoStateMachine_CanTransition(b *testing.B) {
	sm := NewRepoStateMachine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.CanTransition(RepoStatusIdle, RepoStatusBackingUp)
	}
}
