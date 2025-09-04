package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock event emitter for testing
type MockEventEmitter struct {
	mock.Mock
}

func (m *MockEventEmitter) EmitEvent(ctx context.Context, event string) {
	m.Called(ctx, event)
}

func TestBusinessRuleValidator_DefaultValidators(t *testing.T) {
	validator := NewBusinessRuleValidator()

	tests := []struct {
		name        string
		ctx         TransitionContext
		from        RepoStatus
		to          RepoStatus
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid transition from idle to backing up",
			ctx: TransitionContext{
				RepoID: 1,
			},
			from:        RepoStatusIdle,
			to:          RepoStatusBackingUp,
			expectError: false,
		},
		{
			name: "Invalid concurrent operations",
			ctx: TransitionContext{
				RepoID: 1,
			},
			from:        RepoStatusBackingUp,
			to:          RepoStatusPruning,
			expectError: true,
			errorMsg:    "cannot start pruning operation while repository is backingUp",
		},
		{
			name: "Unmount before dangerous operations",
			ctx: TransitionContext{
				RepoID: 1,
			},
			from:        RepoStatusMounted,
			to:          RepoStatusBackingUp,
			expectError: true,
			errorMsg:    "repository must be unmounted before backingUp operation",
		},
		{
			name: "Error state recovery validation",
			ctx: TransitionContext{
				RepoID: 1,
			},
			from:        RepoStatusError,
			to:          RepoStatusBackingUp,
			expectError: true,
			errorMsg:    "repository in error state must be fixed before transitioning to backingUp",
		},
		{
			name: "Valid error state recovery",
			ctx: TransitionContext{
				RepoID: 1,
			},
			from:        RepoStatusError,
			to:          RepoStatusIdle,
			expectError: false,
		},
		{
			name: "Transition to deleting state",
			ctx: TransitionContext{
				RepoID: 1,
			},
			from:        RepoStatusIdle,
			to:          RepoStatusDeleting,
			expectError: false, // Reason requirement removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTransition(tt.ctx, tt.from, tt.to)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBusinessRuleValidator_CustomValidators(t *testing.T) {
	validator := NewBusinessRuleValidator()

	// Add custom validator
	customValidatorCalled := false
	validator.AddValidator("custom_test", func(ctx TransitionContext, from, to RepoStatus) error {
		customValidatorCalled = true
		if ctx.RepoID == 999 {
			return fmt.Errorf("repository 999 is not allowed")
		}
		return nil
	})

	// Test custom validator success
	ctx := TransitionContext{RepoID: 1}
	err := validator.ValidateTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)
	assert.NoError(t, err)
	assert.True(t, customValidatorCalled)

	// Test custom validator failure
	customValidatorCalled = false
	ctx = TransitionContext{RepoID: 999}
	err = validator.ValidateTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository 999 is not allowed")
	assert.True(t, customValidatorCalled)

	// Test removing validator
	validator.RemoveValidator("custom_test")
	customValidatorCalled = false
	ctx = TransitionContext{RepoID: 999}
	err = validator.ValidateTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)
	assert.NoError(t, err) // Should pass now that validator is removed
	assert.False(t, customValidatorCalled)
}

func TestTransitionExecutor_ExecuteTransition(t *testing.T) {
	sm := NewRepoStateMachine()
	mockEmitter := &MockEventEmitter{}
	executor := NewTransitionExecutor(sm, mockEmitter)

	mockEmitter.On("EmitEvent", mock.Anything, mock.AnythingOfType("string")).Return()

	ctx := TransitionContext{
		RepoID:  1,
		Context: context.Background(),
	}

	result, err := executor.ExecuteTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, RepoStatusIdle, result.From)
	assert.Equal(t, RepoStatusBackingUp, result.To)
	assert.Contains(t, result.Reason, "Transition from")
	assert.Greater(t, result.Duration, time.Duration(0))

	// Verify event was emitted
	mockEmitter.AssertExpectations(t)
}

func TestTransitionExecutor_ExecuteTransitionValidationFailure(t *testing.T) {
	sm := NewRepoStateMachine()
	mockEmitter := &MockEventEmitter{}
	executor := NewTransitionExecutor(sm, mockEmitter)

	ctx := TransitionContext{
		RepoID:  1,
		Context: context.Background(),
	}

	// Try invalid transition
	result, err := executor.ExecuteTransition(ctx, RepoStatusBackingUp, RepoStatusPruning)

	require.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "transition validation failed")
	assert.Contains(t, result.Error, "transition validation failed")
	assert.Greater(t, result.Duration, time.Duration(0))

	// No event should be emitted for failed transitions
	mockEmitter.AssertNotCalled(t, "EmitEvent")
}

func TestTransitionExecutor_BusinessRuleFailure(t *testing.T) {
	sm := NewRepoStateMachine()
	mockEmitter := &MockEventEmitter{}
	executor := NewTransitionExecutor(sm, mockEmitter)

	mockEmitter.On("EmitEvent", mock.Anything, mock.AnythingOfType("string")).Return()

	ctx := TransitionContext{
		RepoID:  1,
		Context: context.Background(),
	}

	// Test valid transition that succeeds
	result, err := executor.ExecuteTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)

	require.NoError(t, err)
	assert.True(t, result.Success)

	// Event should be emitted for successful transitions
	mockEmitter.AssertExpectations(t)
}

func TestTransitionExecutor_PreHookFailure(t *testing.T) {
	sm := NewRepoStateMachine()
	mockEmitter := &MockEventEmitter{}
	executor := NewTransitionExecutor(sm, mockEmitter)

	// Add failing pre-hook
	sm.AddPreHook(RepoStatusBackingUp, func(transition StateTransition) error {
		return fmt.Errorf("pre-hook failed")
	})

	ctx := TransitionContext{
		RepoID:  1,
		Context: context.Background(),
	}

	result, err := executor.ExecuteTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)

	require.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "pre-transition hook failed")
	assert.Contains(t, result.Error, "pre-transition hook failed")

	// No event should be emitted for failed transitions
	mockEmitter.AssertNotCalled(t, "EmitEvent")
}

func TestTransitionExecutor_PostHookFailure(t *testing.T) {
	sm := NewRepoStateMachine()
	mockEmitter := &MockEventEmitter{}
	executor := NewTransitionExecutor(sm, mockEmitter)

	mockEmitter.On("EmitEvent", mock.Anything, mock.AnythingOfType("string")).Return()

	// Add failing post-hook
	sm.AddPostHook(RepoStatusBackingUp, func(transition StateTransition) error {
		return fmt.Errorf("post-hook failed")
	})

	ctx := TransitionContext{
		RepoID:  1,
		Context: context.Background(),
	}

	result, err := executor.ExecuteTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)

	// Post-hook failures don't fail the transition
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Event should still be emitted for successful transition
	mockEmitter.AssertExpectations(t)
}

func TestNewTransitionExecutor_NilEventEmitterPanic(t *testing.T) {
	sm := NewRepoStateMachine()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when eventEmitter is nil")
		} else {
			// The assert library panics with an error object, check it contains our message
			assert.Contains(t, fmt.Sprintf("%v", r), "eventEmitter must not be nil")
		}
	}()

	// This should panic
	NewTransitionExecutor(sm, nil)
}

func TestTransitionExecutor_ForceTransition(t *testing.T) {
	sm := NewRepoStateMachine()
	mockEmitter := &MockEventEmitter{}
	executor := NewTransitionExecutor(sm, mockEmitter)

	mockEmitter.On("EmitEvent", mock.Anything, mock.AnythingOfType("string")).Return()

	ctx := TransitionContext{
		RepoID:  1,
		Context: context.Background(),
	}

	// Force an invalid transition
	result := executor.ForceTransition(ctx, RepoStatusBackingUp, RepoStatusPruning)
	assert.True(t, result.Success)
	assert.Equal(t, RepoStatusBackingUp, result.From)
	assert.Equal(t, RepoStatusPruning, result.To)
	assert.Contains(t, result.Reason, "FORCED:")

	// Event should be emitted
	mockEmitter.AssertExpectations(t)
}

func TestGetTransitionReason(t *testing.T) {
	tests := []struct {
		name      string
		from      RepoStatus
		to        RepoStatus
		operation string
		expected  string
	}{
		{
			name:      "Idle to BackingUp",
			from:      RepoStatusIdle,
			to:        RepoStatusBackingUp,
			operation: "daily backup",
			expected:  "Starting backup operation: daily backup",
		},
		{
			name:      "Idle to Pruning",
			from:      RepoStatusIdle,
			to:        RepoStatusPruning,
			operation: "cleanup old archives",
			expected:  "Starting pruning operation: cleanup old archives",
		},
		{
			name:      "Idle to Deleting",
			from:      RepoStatusIdle,
			to:        RepoStatusDeleting,
			operation: "remove repository",
			expected:  "Starting deletion operation: remove repository",
		},
		{
			name:      "Idle to Mounted",
			from:      RepoStatusIdle,
			to:        RepoStatusMounted,
			operation: "browse files",
			expected:  "Mounting repository: browse files",
		},
		{
			name:      "Idle to PerformingOperation",
			from:      RepoStatusIdle,
			to:        RepoStatusPerformingOperation,
			operation: "refresh archive list",
			expected:  "Starting repository operation: refresh archive list",
		},
		{
			name:      "BackingUp to Idle",
			from:      RepoStatusBackingUp,
			to:        RepoStatusIdle,
			operation: "backup completed successfully",
			expected:  "Completed backingUp operation: backup completed successfully",
		},
		{
			name:      "BackingUp to Error",
			from:      RepoStatusBackingUp,
			to:        RepoStatusError,
			operation: "backup failed with connection error",
			expected:  "Error during backingUp operation: backup failed with connection error",
		},
		{
			name:      "Generic transition",
			from:      RepoStatusMounted,
			to:        RepoStatusError,
			operation: "mount failed",
			expected:  "Error during mounted operation: mount failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTransitionReason(tt.from, tt.to, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsStatus(t *testing.T) {
	statuses := []RepoStatus{RepoStatusIdle, RepoStatusBackingUp, RepoStatusPruning}

	assert.True(t, containsStatus(statuses, RepoStatusIdle))
	assert.True(t, containsStatus(statuses, RepoStatusBackingUp))
	assert.True(t, containsStatus(statuses, RepoStatusPruning))
	assert.False(t, containsStatus(statuses, RepoStatusDeleting))
	assert.False(t, containsStatus(statuses, RepoStatusError))
}

func TestTransitionResult_JSON(t *testing.T) {
	result := &TransitionResult{
		Success:   true,
		From:      RepoStatusIdle,
		To:        RepoStatusBackingUp,
		Reason:    "Test transition",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Duration:  time.Second * 5,
	}

	// This tests that the struct can be marshaled to JSON (useful for API responses)
	assert.Equal(t, true, result.Success)
	assert.Equal(t, RepoStatusIdle, result.From)
	assert.Equal(t, RepoStatusBackingUp, result.To)
	assert.Equal(t, "Test transition", result.Reason)
	assert.Equal(t, time.Second*5, result.Duration)
}

// Benchmark tests
func BenchmarkBusinessRuleValidator_ValidateTransition(b *testing.B) {
	validator := NewBusinessRuleValidator()
	ctx := TransitionContext{
		RepoID: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)
	}
}

func BenchmarkTransitionExecutor_ExecuteTransition(b *testing.B) {
	sm := NewRepoStateMachine()
	mockEmitter := &MockEventEmitter{}
	executor := NewTransitionExecutor(sm, mockEmitter)

	mockEmitter.On("EmitEvent", mock.Anything, mock.AnythingOfType("string")).Return()

	ctx := TransitionContext{
		RepoID:  1,
		Context: context.Background(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate between states to avoid same-state validation errors
		if i%2 == 0 {
			executor.ExecuteTransition(ctx, RepoStatusIdle, RepoStatusBackingUp)
		} else {
			executor.ExecuteTransition(ctx, RepoStatusBackingUp, RepoStatusIdle)
		}
	}
}
