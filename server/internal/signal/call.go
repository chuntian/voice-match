package signal

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// CallState represents the state of a call.
type CallState string

const (
	CallStateIdle      CallState = "idle"
	CallStateRinging   CallState = "ringing"
	CallStateConnected CallState = "connected"
	CallStateEnded     CallState = "ended"
)

// CallSession holds all runtime information for a single call.
type CallSession struct {
	CallID        string
	CallerID      string
	CalleeID      string
	State         CallState
	RoomName      string
	RtcToken      string
	StartTime     time.Time
	CreatedAt     time.Time

	mu            sync.Mutex
	ringTimer     *time.Timer
}

// CallManager tracks all active calls and enforces the state machine.
type CallManager struct {
	mu      sync.RWMutex
	calls   map[string]*CallSession // callID -> session

	// onRingTimeout is invoked when a ringing call times out without
	// being answered. The handler should notify both parties.
	onRingTimeout func(callID string)

	// ringTimeoutDuration is how long a call can stay in ringing.
	ringTimeoutDuration time.Duration
}

// NewCallManager creates a CallManager with default settings.
func NewCallManager() *CallManager {
	return &CallManager{
		calls:               make(map[string]*CallSession),
		ringTimeoutDuration: 30 * time.Second,
	}
}

// SetRingTimeoutDuration overrides the ringing timeout.
func (cm *CallManager) SetRingTimeoutDuration(d time.Duration) {
	cm.ringTimeoutDuration = d
}

// SetOnRingTimeout registers the callback for ring timeouts.
func (cm *CallManager) SetOnRingTimeout(fn func(callID string)) {
	cm.onRingTimeout = fn
}

// Invite creates a new call session in ringing state.
// Returns an error if either party is already in a call.
func (cm *CallManager) Invite(callID, callerID, calleeID, roomName, rtcToken string) (*CallSession, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check caller is not already in a call.
	for _, c := range cm.calls {
		if c.State == CallStateEnded {
			continue
		}
		if c.CallerID == callerID || c.CalleeID == callerID {
			return nil, fmt.Errorf("%w: caller %s already in call", ErrAlreadyInCall, callerID)
		}
		if c.CallerID == calleeID || c.CalleeID == calleeID {
			return nil, fmt.Errorf("%w: callee %s already in call", ErrAlreadyInCall, calleeID)
		}
	}

	now := time.Now()
	sess := &CallSession{
		CallID:    callID,
		CallerID:  callerID,
		CalleeID:  calleeID,
		State:     CallStateRinging,
		RoomName:  roomName,
		RtcToken:  rtcToken,
		CreatedAt: now,
	}

	// Set up ring timeout.
	sess.mu.Lock()
	sess.ringTimer = time.AfterFunc(cm.ringTimeoutDuration, func() {
		log.Printf("[call] ring timeout callID=%s", callID)
		if err := cm.Cancel(callID, "ring_timeout"); err != nil {
			log.Printf("[call] cancel after ring timeout failed: %v", err)
		}
		if cm.onRingTimeout != nil {
			cm.onRingTimeout(callID)
		}
	})
	sess.mu.Unlock()

	cm.calls[callID] = sess
	log.Printf("[call] invite callID=%s caller=%s callee=%s", callID, callerID, calleeID)
	return sess, nil
}

// Accept transitions a ringing call to connected.
func (cm *CallManager) Accept(callID, userID string) (*CallSession, error) {
	cm.mu.RLock()
	sess, ok := cm.calls[callID]
	cm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: call %s", ErrCallNotFound, callID)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.State != CallStateRinging {
		return nil, fmt.Errorf("invalid transition: %s -> connected (callID=%s)", sess.State, callID)
	}
	if userID != sess.CalleeID {
		return nil, fmt.Errorf("only callee can accept (callID=%s)", callID)
	}

	sess.State = CallStateConnected
	sess.StartTime = time.Now()
	if sess.ringTimer != nil {
		sess.ringTimer.Stop()
		sess.ringTimer = nil
	}

	log.Printf("[call] accept callID=%s user=%s", callID, userID)
	return sess, nil
}

// Reject transitions a ringing call to ended (rejected by callee).
func (cm *CallManager) Reject(callID, userID, reason string) (*CallSession, error) {
	cm.mu.RLock()
	sess, ok := cm.calls[callID]
	cm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: call %s", ErrCallNotFound, callID)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.State != CallStateRinging {
		return nil, fmt.Errorf("invalid transition: %s -> ended(reject) (callID=%s)", sess.State, callID)
	}
	if userID != sess.CalleeID {
		return nil, fmt.Errorf("only callee can reject (callID=%s)", callID)
	}

	sess.State = CallStateEnded
	if sess.ringTimer != nil {
		sess.ringTimer.Stop()
		sess.ringTimer = nil
	}

	log.Printf("[call] reject callID=%s user=%s reason=%s", callID, userID, reason)
	return sess, nil
}

// Cancel transitions a ringing call to ended (canceled by caller).
func (cm *CallManager) Cancel(callID, reason string) error {
	cm.mu.RLock()
	sess, ok := cm.calls[callID]
	cm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("%w: call %s", ErrCallNotFound, callID)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.State != CallStateRinging {
		return fmt.Errorf("invalid transition: %s -> ended(cancel) (callID=%s)", sess.State, callID)
	}

	sess.State = CallStateEnded
	if sess.ringTimer != nil {
		sess.ringTimer.Stop()
		sess.ringTimer = nil
	}

	log.Printf("[call] cancel callID=%s reason=%s", callID, reason)
	return nil
}

// End transitions a connected call to ended.
func (cm *CallManager) End(callID, userID string, reason string) (*CallSession, int, error) {
	cm.mu.RLock()
	sess, ok := cm.calls[callID]
	cm.mu.RUnlock()

	if !ok {
		return nil, 0, fmt.Errorf("%w: call %s", ErrCallNotFound, callID)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.State != CallStateConnected {
		return nil, 0, fmt.Errorf("invalid transition: %s -> ended(end) (callID=%s)", sess.State, callID)
	}

	sess.State = CallStateEnded
	duration := int(time.Since(sess.StartTime).Seconds())

	log.Printf("[call] end callID=%s user=%s duration=%ds reason=%s", callID, userID, duration, reason)
	return sess, duration, nil
}

// Get returns the call session for the given callID.
func (cm *CallManager) Get(callID string) (*CallSession, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	sess, ok := cm.calls[callID]
	return sess, ok
}

// GetByUser returns the active (non-ended) call for a user, if any.
func (cm *CallManager) GetByUser(userID string) (*CallSession, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for _, c := range cm.calls {
		if c.State == CallStateEnded {
			continue
		}
		if c.CallerID == userID || c.CalleeID == userID {
			return c, true
		}
	}
	return nil, false
}

// Remove deletes a call session from the manager. Called after all
// participants have been notified and the call is fully cleaned up.
func (cm *CallManager) Remove(callID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.calls, callID)
}

// Count returns the number of active (non-ended) calls.
func (cm *CallManager) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	n := 0
	for _, c := range cm.calls {
		if c.State != CallStateEnded {
			n++
		}
	}
	return n
}

// ---- Errors ----

var (
	ErrAlreadyInCall = errors.New("already in call")
	ErrCallNotFound  = errors.New("call not found")
)
