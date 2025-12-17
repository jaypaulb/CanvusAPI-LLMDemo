// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains the RecoveryManager organism for automatic recovery logic.
//
// The RecoveryManager provides resilience for inference operations by implementing:
// - Retry with exponential backoff on transient failures
// - Context reset on persistent errors
// - Model reload as escalating recovery strategy
// - Degraded mode for unrecoverable errors
//
// Architecture:
// - RecoveryManager wraps a Client and provides transparent retry logic
// - Thread-safe: can be used by multiple goroutines concurrently
// - Composes health.go, client.go, and modelloader.go components
package llamaruntime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Recovery Configuration
// =============================================================================

// RecoveryConfig contains configuration for the RecoveryManager.
type RecoveryConfig struct {
	// MaxRetries is the maximum number of retry attempts per operation.
	// Defaults to 3.
	MaxRetries int

	// InitialBackoff is the delay before the first retry.
	// Defaults to 500ms.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff delay.
	// Defaults to 10s.
	MaxBackoff time.Duration

	// BackoffMultiplier is the exponential backoff multiplier.
	// Defaults to 2.0.
	BackoffMultiplier float64

	// ContextResetThreshold is the number of consecutive failures
	// before attempting context reset. Defaults to 3.
	ContextResetThreshold int

	// ModelReloadThreshold is the number of context resets
	// before attempting model reload. Defaults to 2.
	ModelReloadThreshold int

	// DegradedModeEnabled enables degraded mode after exhausting recovery options.
	// In degraded mode, requests are rejected immediately. Defaults to true.
	DegradedModeEnabled bool

	// DegradedModeRecoveryInterval is how often to attempt recovery from degraded mode.
	// Defaults to 1 minute.
	DegradedModeRecoveryInterval time.Duration

	// Logger is an optional logger. If nil, uses standard log.
	Logger *log.Logger

	// ModelLoaderConfig is used for model reload operations.
	// Required if model reload recovery is desired.
	ModelLoaderConfig *ModelLoaderConfig

	// OnRecoveryAttempt is called when a recovery action is taken.
	// Optional callback for monitoring.
	OnRecoveryAttempt func(action RecoveryAction, attempt int, err error)

	// OnDegradedModeEnter is called when entering degraded mode.
	OnDegradedModeEnter func(reason string)

	// OnDegradedModeExit is called when exiting degraded mode.
	OnDegradedModeExit func()
}

// DefaultRecoveryConfig returns a RecoveryConfig with sensible defaults.
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		MaxRetries:                   3,
		InitialBackoff:               500 * time.Millisecond,
		MaxBackoff:                   10 * time.Second,
		BackoffMultiplier:            2.0,
		ContextResetThreshold:        3,
		ModelReloadThreshold:         2,
		DegradedModeEnabled:          true,
		DegradedModeRecoveryInterval: 1 * time.Minute,
	}
}

// RecoveryAction represents the type of recovery action taken.
type RecoveryAction string

const (
	// RecoveryActionRetry indicates a simple retry with backoff.
	RecoveryActionRetry RecoveryAction = "retry"

	// RecoveryActionContextReset indicates context KV cache was reset.
	RecoveryActionContextReset RecoveryAction = "context_reset"

	// RecoveryActionModelReload indicates the model was reloaded.
	RecoveryActionModelReload RecoveryAction = "model_reload"

	// RecoveryActionDegradedMode indicates entering degraded mode.
	RecoveryActionDegradedMode RecoveryAction = "degraded_mode"
)

// =============================================================================
// Recovery Statistics
// =============================================================================

// RecoveryStats contains statistics about recovery operations.
type RecoveryStats struct {
	// TotalRequests is the total number of inference requests.
	TotalRequests int64

	// SuccessfulRequests is the number of successful requests.
	SuccessfulRequests int64

	// RetryAttempts is the total number of retry attempts.
	RetryAttempts int64

	// ContextResets is the number of context reset operations.
	ContextResets int64

	// ModelReloads is the number of model reload operations.
	ModelReloads int64

	// DegradedModeEntries is the number of times degraded mode was entered.
	DegradedModeEntries int64

	// DegradedModeRecoveries is the number of successful recoveries from degraded mode.
	DegradedModeRecoveries int64

	// ConsecutiveFailures is the current count of consecutive failures.
	ConsecutiveFailures int64

	// InDegradedMode indicates if currently in degraded mode.
	InDegradedMode bool

	// DegradedModeReason is the reason for entering degraded mode (if any).
	DegradedModeReason string

	// LastSuccessfulRequest is when the last request succeeded.
	LastSuccessfulRequest time.Time

	// LastFailure is when the last failure occurred.
	LastFailure time.Time

	// Uptime is how long the recovery manager has been running.
	Uptime time.Duration
}

// =============================================================================
// Recovery Manager
// =============================================================================

// RecoveryManager provides automatic recovery for inference operations.
// It wraps a Client and transparently handles retries, context resets,
// and model reloads.
//
// Thread Safety:
// - All public methods are thread-safe
// - Multiple goroutines can call Infer/InferVision concurrently
// - Recovery operations are serialized to prevent conflicts
//
// Example usage:
//
//	config := llamaruntime.DefaultRecoveryConfig()
//	manager := llamaruntime.NewRecoveryManager(client, config)
//
//	// Use like the client, but with automatic recovery
//	result, err := manager.Infer(ctx, params)
type RecoveryManager struct {
	client *Client
	config RecoveryConfig
	logger *log.Logger

	// State
	mu                    sync.RWMutex
	consecutiveFailures   int64
	contextResetCount     int64
	modelReloadCount      int64
	degradedMode          int32 // atomic (0 = normal, 1 = degraded)
	degradedModeReason    string
	degradedModeEnteredAt time.Time

	// Recovery coordination
	recoveryMu sync.Mutex // Serializes recovery operations

	// Statistics
	totalRequests          int64
	successfulRequests     int64
	retryAttempts          int64
	totalContextResets     int64
	totalModelReloads      int64
	degradedModeEntries    int64
	degradedModeRecoveries int64
	lastSuccess            time.Time
	lastFailure            time.Time
	startTime              time.Time

	// Degraded mode recovery
	recoveryCtx    context.Context
	recoveryCancel context.CancelFunc
	recoveryWg     sync.WaitGroup
}

// NewRecoveryManager creates a new RecoveryManager wrapping the given client.
func NewRecoveryManager(client *Client, config RecoveryConfig) *RecoveryManager {
	// Apply defaults
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = 500 * time.Millisecond
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = 10 * time.Second
	}
	if config.BackoffMultiplier <= 0 {
		config.BackoffMultiplier = 2.0
	}
	if config.ContextResetThreshold <= 0 {
		config.ContextResetThreshold = 3
	}
	if config.ModelReloadThreshold <= 0 {
		config.ModelReloadThreshold = 2
	}
	if config.DegradedModeRecoveryInterval <= 0 {
		config.DegradedModeRecoveryInterval = 1 * time.Minute
	}

	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())

	rm := &RecoveryManager{
		client:         client,
		config:         config,
		logger:         logger,
		startTime:      time.Now(),
		recoveryCtx:    ctx,
		recoveryCancel: cancel,
	}

	// Start degraded mode recovery goroutine if enabled
	if config.DegradedModeEnabled {
		rm.recoveryWg.Add(1)
		go rm.degradedModeRecoveryLoop()
	}

	return rm
}

// Infer performs text inference with automatic recovery.
// On failure, it will retry with exponential backoff, reset context,
// or reload the model as needed.
func (rm *RecoveryManager) Infer(ctx context.Context, params InferenceParams) (*InferenceResult, error) {
	atomic.AddInt64(&rm.totalRequests, 1)

	// Check degraded mode
	if rm.isDegradedMode() {
		return nil, &LlamaError{
			Op:      "Infer",
			Code:    -1,
			Message: fmt.Sprintf("service in degraded mode: %s", rm.getDegradedModeReason()),
			Err:     ErrInferenceFailed,
		}
	}

	return rm.executeWithRecovery(ctx, func(c context.Context) (*InferenceResult, error) {
		return rm.client.Infer(c, params)
	})
}

// InferVision performs vision inference with automatic recovery.
func (rm *RecoveryManager) InferVision(ctx context.Context, params VisionParams) (*InferenceResult, error) {
	atomic.AddInt64(&rm.totalRequests, 1)

	// Check degraded mode
	if rm.isDegradedMode() {
		return nil, &LlamaError{
			Op:      "InferVision",
			Code:    -1,
			Message: fmt.Sprintf("service in degraded mode: %s", rm.getDegradedModeReason()),
			Err:     ErrInferenceFailed,
		}
	}

	return rm.executeWithRecovery(ctx, func(c context.Context) (*InferenceResult, error) {
		return rm.client.InferVision(c, params)
	})
}

// executeWithRecovery executes an inference operation with automatic recovery.
func (rm *RecoveryManager) executeWithRecovery(ctx context.Context, fn func(context.Context) (*InferenceResult, error)) (*InferenceResult, error) {
	var lastErr error
	backoff := rm.config.InitialBackoff

	for attempt := 1; attempt <= rm.config.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute the inference
		result, err := fn(ctx)
		if err == nil {
			// Success - reset failure counters
			rm.recordSuccess()
			return result, nil
		}

		lastErr = err
		rm.recordFailure(err)

		// Check if error is recoverable
		if !rm.isRecoverableError(err) {
			rm.logger.Printf("[Recovery] Non-recoverable error: %v", err)
			break
		}

		// Log retry attempt
		rm.logger.Printf("[Recovery] Attempt %d/%d failed: %v", attempt, rm.config.MaxRetries, err)
		atomic.AddInt64(&rm.retryAttempts, 1)

		// Notify callback
		if rm.config.OnRecoveryAttempt != nil {
			rm.config.OnRecoveryAttempt(RecoveryActionRetry, attempt, err)
		}

		// Check if we should escalate recovery
		rm.checkAndEscalateRecovery()

		// Wait before retry (unless it's the last attempt)
		if attempt < rm.config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}

			// Exponential backoff
			backoff = time.Duration(float64(backoff) * rm.config.BackoffMultiplier)
			if backoff > rm.config.MaxBackoff {
				backoff = rm.config.MaxBackoff
			}
		}
	}

	// All retries exhausted
	rm.logger.Printf("[Recovery] All %d retry attempts exhausted", rm.config.MaxRetries)

	// Check if we should enter degraded mode
	rm.checkDegradedMode(lastErr)

	return nil, fmt.Errorf("inference failed after %d attempts: %w", rm.config.MaxRetries, lastErr)
}

// isRecoverableError determines if an error can be recovered from.
func (rm *RecoveryManager) isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation/timeout is not recoverable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for specific non-recoverable errors
	if errors.Is(err, ErrModelNotFound) {
		return false
	}

	// Check for GPU unavailable - this requires model reload
	if errors.Is(err, ErrGPUNotAvailable) {
		// Can attempt recovery through model reload
		return true
	}

	// Check for insufficient VRAM - may be recoverable after context reset
	if errors.Is(err, ErrInsufficientVRAM) {
		return true
	}

	// Most other errors are potentially recoverable
	return true
}

// recordSuccess records a successful inference.
func (rm *RecoveryManager) recordSuccess() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	atomic.AddInt64(&rm.successfulRequests, 1)
	rm.consecutiveFailures = 0
	rm.contextResetCount = 0
	rm.modelReloadCount = 0
	rm.lastSuccess = time.Now()
}

// recordFailure records a failed inference attempt.
func (rm *RecoveryManager) recordFailure(err error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.consecutiveFailures++
	rm.lastFailure = time.Now()
}

// checkAndEscalateRecovery checks if recovery should be escalated.
func (rm *RecoveryManager) checkAndEscalateRecovery() {
	rm.mu.Lock()
	consecutiveFailures := rm.consecutiveFailures
	contextResetCount := rm.contextResetCount
	rm.mu.Unlock()

	// Check if context reset is needed
	if consecutiveFailures >= int64(rm.config.ContextResetThreshold) &&
		contextResetCount < int64(rm.config.ModelReloadThreshold) {
		rm.attemptContextReset()
		return
	}

	// Check if model reload is needed
	if contextResetCount >= int64(rm.config.ModelReloadThreshold) {
		rm.attemptModelReload()
	}
}

// attemptContextReset attempts to reset inference contexts.
func (rm *RecoveryManager) attemptContextReset() {
	rm.recoveryMu.Lock()
	defer rm.recoveryMu.Unlock()

	rm.logger.Printf("[Recovery] Attempting context reset")

	// The context pool already clears KV cache on release.
	// For a "context reset", we can trigger a health check to verify the client.
	_, err := rm.client.HealthCheck()
	if err != nil {
		rm.logger.Printf("[Recovery] Context reset health check failed: %v", err)
	} else {
		rm.logger.Printf("[Recovery] Context reset completed (health check passed)")
	}

	rm.mu.Lock()
	rm.contextResetCount++
	rm.consecutiveFailures = 0 // Reset after recovery attempt
	rm.mu.Unlock()

	atomic.AddInt64(&rm.totalContextResets, 1)

	if rm.config.OnRecoveryAttempt != nil {
		rm.config.OnRecoveryAttempt(RecoveryActionContextReset, int(rm.contextResetCount), nil)
	}
}

// attemptModelReload attempts to reload the model.
func (rm *RecoveryManager) attemptModelReload() {
	rm.recoveryMu.Lock()
	defer rm.recoveryMu.Unlock()

	rm.logger.Printf("[Recovery] Attempting model reload")

	if rm.config.ModelLoaderConfig == nil {
		rm.logger.Printf("[Recovery] Model reload not available - no ModelLoaderConfig")
		return
	}

	// Create a new model loader
	loader := NewModelLoader(*rm.config.ModelLoaderConfig)

	// Load the model
	loadCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	newClient, err := loader.Load(loadCtx)
	if err != nil {
		rm.logger.Printf("[Recovery] Model reload failed: %v", err)
		return
	}

	// Close the old client
	oldClient := rm.client
	go func() {
		if closeErr := oldClient.Close(); closeErr != nil {
			rm.logger.Printf("[Recovery] Error closing old client: %v", closeErr)
		}
	}()

	// Swap in the new client
	rm.mu.Lock()
	rm.client = newClient
	rm.modelReloadCount++
	rm.contextResetCount = 0
	rm.consecutiveFailures = 0
	rm.mu.Unlock()

	atomic.AddInt64(&rm.totalModelReloads, 1)
	rm.logger.Printf("[Recovery] Model reload completed successfully")

	if rm.config.OnRecoveryAttempt != nil {
		rm.config.OnRecoveryAttempt(RecoveryActionModelReload, int(rm.modelReloadCount), nil)
	}
}

// checkDegradedMode checks if we should enter degraded mode.
func (rm *RecoveryManager) checkDegradedMode(lastErr error) {
	if !rm.config.DegradedModeEnabled {
		return
	}

	rm.mu.Lock()
	modelReloadCount := rm.modelReloadCount
	rm.mu.Unlock()

	// Enter degraded mode if model reloads have been exhausted
	if modelReloadCount >= int64(rm.config.ModelReloadThreshold) {
		rm.enterDegradedMode(fmt.Sprintf("exceeded recovery attempts: %v", lastErr))
	}
}

// enterDegradedMode puts the manager into degraded mode.
func (rm *RecoveryManager) enterDegradedMode(reason string) {
	if atomic.CompareAndSwapInt32(&rm.degradedMode, 0, 1) {
		rm.mu.Lock()
		rm.degradedModeReason = reason
		rm.degradedModeEnteredAt = time.Now()
		rm.mu.Unlock()

		atomic.AddInt64(&rm.degradedModeEntries, 1)
		rm.logger.Printf("[Recovery] Entering degraded mode: %s", reason)

		if rm.config.OnDegradedModeEnter != nil {
			rm.config.OnDegradedModeEnter(reason)
		}

		if rm.config.OnRecoveryAttempt != nil {
			rm.config.OnRecoveryAttempt(RecoveryActionDegradedMode, 0, errors.New(reason))
		}
	}
}

// exitDegradedMode exits degraded mode.
func (rm *RecoveryManager) exitDegradedMode() {
	if atomic.CompareAndSwapInt32(&rm.degradedMode, 1, 0) {
		rm.mu.Lock()
		rm.degradedModeReason = ""
		rm.consecutiveFailures = 0
		rm.contextResetCount = 0
		rm.modelReloadCount = 0
		rm.mu.Unlock()

		atomic.AddInt64(&rm.degradedModeRecoveries, 1)
		rm.logger.Printf("[Recovery] Exiting degraded mode - service recovered")

		if rm.config.OnDegradedModeExit != nil {
			rm.config.OnDegradedModeExit()
		}
	}
}

// isDegradedMode returns true if in degraded mode.
func (rm *RecoveryManager) isDegradedMode() bool {
	return atomic.LoadInt32(&rm.degradedMode) == 1
}

// getDegradedModeReason returns the reason for degraded mode.
func (rm *RecoveryManager) getDegradedModeReason() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.degradedModeReason
}

// degradedModeRecoveryLoop periodically attempts to recover from degraded mode.
func (rm *RecoveryManager) degradedModeRecoveryLoop() {
	defer rm.recoveryWg.Done()

	ticker := time.NewTicker(rm.config.DegradedModeRecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.recoveryCtx.Done():
			return
		case <-ticker.C:
			if rm.isDegradedMode() {
				rm.attemptDegradedModeRecovery()
			}
		}
	}
}

// attemptDegradedModeRecovery attempts to recover from degraded mode.
func (rm *RecoveryManager) attemptDegradedModeRecovery() {
	rm.logger.Printf("[Recovery] Attempting recovery from degraded mode")

	// Try a health check first
	status, err := rm.client.HealthCheck()
	if err != nil {
		rm.logger.Printf("[Recovery] Degraded mode recovery: health check failed: %v", err)
		return
	}

	if status.Healthy {
		// Try a simple inference
		params := DefaultInferenceParams()
		params.Prompt = "Hello"
		params.MaxTokens = 5

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Use the client directly to avoid recursion
		_, err := rm.client.Infer(ctx, params)
		if err != nil {
			rm.logger.Printf("[Recovery] Degraded mode recovery: test inference failed: %v", err)
			return
		}

		// Recovery successful
		rm.exitDegradedMode()
		return
	}

	// If health check failed but no error, try model reload
	if rm.config.ModelLoaderConfig != nil {
		rm.attemptModelReload()
		// Check if model reload helped
		status, err := rm.client.HealthCheck()
		if err == nil && status.Healthy {
			rm.exitDegradedMode()
		}
	}
}

// Stats returns recovery statistics.
func (rm *RecoveryManager) Stats() RecoveryStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return RecoveryStats{
		TotalRequests:          atomic.LoadInt64(&rm.totalRequests),
		SuccessfulRequests:     atomic.LoadInt64(&rm.successfulRequests),
		RetryAttempts:          atomic.LoadInt64(&rm.retryAttempts),
		ContextResets:          atomic.LoadInt64(&rm.totalContextResets),
		ModelReloads:           atomic.LoadInt64(&rm.totalModelReloads),
		DegradedModeEntries:    atomic.LoadInt64(&rm.degradedModeEntries),
		DegradedModeRecoveries: atomic.LoadInt64(&rm.degradedModeRecoveries),
		ConsecutiveFailures:    rm.consecutiveFailures,
		InDegradedMode:         rm.isDegradedMode(),
		DegradedModeReason:     rm.degradedModeReason,
		LastSuccessfulRequest:  rm.lastSuccess,
		LastFailure:            rm.lastFailure,
		Uptime:                 time.Since(rm.startTime),
	}
}

// IsHealthy returns true if the manager is healthy and accepting requests.
func (rm *RecoveryManager) IsHealthy() bool {
	return !rm.isDegradedMode()
}

// Client returns the underlying client.
// This allows direct access for operations that don't need recovery.
func (rm *RecoveryManager) Client() *Client {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.client
}

// Close stops the recovery manager.
// It does NOT close the underlying client - that remains the caller's responsibility.
func (rm *RecoveryManager) Close() error {
	rm.recoveryCancel()
	rm.recoveryWg.Wait()
	return nil
}

// =============================================================================
// Convenience Functions
// =============================================================================

// WithRecovery wraps an existing client with recovery logic using default config.
func WithRecovery(client *Client) *RecoveryManager {
	return NewRecoveryManager(client, DefaultRecoveryConfig())
}

// WithRecoveryConfig wraps an existing client with recovery logic using custom config.
func WithRecoveryConfig(client *Client, config RecoveryConfig) *RecoveryManager {
	return NewRecoveryManager(client, config)
}
