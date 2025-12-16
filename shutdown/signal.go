package shutdown

import (
	"sync"
)

// SignalCounter tracks repeated shutdown signals and triggers forced shutdown.
//
// This is a molecule that composes atomic counting with a callback mechanism
// to handle the common pattern of "first signal = graceful, second = force".
//
// Usage:
//
//	counter := NewSignalCounter(2, func() {
//	    log.Println("Force shutdown!")
//	    os.Exit(1)
//	})
//
//	// In signal handler:
//	signal.Notify(sigChan, os.Interrupt)
//	go func() {
//	    for range sigChan {
//	        count := counter.Increment()
//	        if count == 1 {
//	            log.Println("Graceful shutdown initiated...")
//	            cancel() // trigger graceful shutdown
//	        }
//	        // Force callback called automatically when threshold reached
//	    }
//	}()
type SignalCounter struct {
	mu         sync.Mutex
	count      int
	forceAfter int
	onForce    func()
}

// NewSignalCounter creates a new SignalCounter.
//
// Parameters:
//   - forceAfter: the count at which onForce will be called (typically 2)
//   - onForce: callback invoked when count reaches forceAfter (may be nil)
//
// Example: NewSignalCounter(2, forceExit) means first signal starts graceful
// shutdown, second signal triggers forceExit.
func NewSignalCounter(forceAfter int, onForce func()) *SignalCounter {
	return &SignalCounter{
		forceAfter: forceAfter,
		onForce:    onForce,
	}
}

// Increment increases the signal count by one and returns the new count.
// If the count reaches or exceeds forceAfter, the onForce callback is invoked.
//
// The callback is invoked while holding the lock, so it should be fast or
// should exit the process. Blocking callbacks will prevent further increments.
func (s *SignalCounter) Increment() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	if s.count >= s.forceAfter && s.onForce != nil {
		s.onForce()
	}
	return s.count
}

// Count returns the current signal count.
func (s *SignalCounter) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

// Reset resets the signal count to zero.
// Useful for testing or if the shutdown was cancelled.
func (s *SignalCounter) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count = 0
}

// SetForceCallback changes the force callback.
// This can be used to update the callback after creation.
func (s *SignalCounter) SetForceCallback(onForce func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onForce = onForce
}
