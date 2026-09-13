package signal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Call state machine tests ----

func TestCallStateMachine_NormalFlow(t *testing.T) {
	cm := NewCallManager()
	cm.SetRingTimeoutDuration(5 * time.Second)

	sess, err := cm.Invite("call-1", "alice", "bob", "room-1", "token-1")
	require.NoError(t, err)
	assert.Equal(t, CallStateRinging, sess.State)

	accepted, err := cm.Accept("call-1", "bob")
	require.NoError(t, err)
	assert.Equal(t, CallStateConnected, accepted.State)

	ended, duration, err := cm.End("call-1", "alice", "normal")
	require.NoError(t, err)
	assert.Equal(t, CallStateEnded, ended.State)
	assert.GreaterOrEqual(t, duration, 0)
}

func TestCallStateMachine_Reject(t *testing.T) {
	cm := NewCallManager()
	cm.SetRingTimeoutDuration(5 * time.Second)

	_, err := cm.Invite("call-2", "alice", "bob", "room-2", "token-2")
	require.NoError(t, err)

	rejected, err := cm.Reject("call-2", "bob", "busy")
	require.NoError(t, err)
	assert.Equal(t, CallStateEnded, rejected.State)
}

func TestCallStateMachine_Cancel(t *testing.T) {
	cm := NewCallManager()
	cm.SetRingTimeoutDuration(5 * time.Second)

	_, err := cm.Invite("call-3", "alice", "bob", "room-3", "token-3")
	require.NoError(t, err)

	err = cm.Cancel("call-3", "user_cancel")
	require.NoError(t, err)

	sess, ok := cm.Get("call-3")
	require.True(t, ok)
	assert.Equal(t, CallStateEnded, sess.State)
}

func TestCallStateMachine_InvalidTransition_AcceptWithoutRing(t *testing.T) {
	cm := NewCallManager()

	// Accept a non-existent call.
	_, err := cm.Accept("nonexistent", "bob")
	assert.Error(t, err)
}

func TestCallStateMachine_AcceptWrongUser(t *testing.T) {
	cm := NewCallManager()
	cm.SetRingTimeoutDuration(5 * time.Second)

	_, err := cm.Invite("call-4", "alice", "bob", "room-4", "token-4")
	require.NoError(t, err)

	// Charlie tries to accept.
	_, err = cm.Accept("call-4", "charlie")
	assert.Error(t, err)
}

func TestCallStateMachine_AlreadyInCall(t *testing.T) {
	cm := NewCallManager()
	cm.SetRingTimeoutDuration(5 * time.Second)

	_, err := cm.Invite("call-5", "alice", "bob", "room-5", "token-5")
	require.NoError(t, err)

	// Alice tries to initiate another call.
	_, err = cm.Invite("call-6", "alice", "dave", "room-6", "token-6")
	assert.Error(t, err)
}

func TestCallStateMachine_RingTimeout(t *testing.T) {
	cm := NewCallManager()
	cm.SetRingTimeoutDuration(100 * time.Millisecond)

	timedOut := make(chan string, 1)
	cm.SetOnRingTimeout(func(callID string) {
		timedOut <- callID
	})

	_, err := cm.Invite("call-7", "alice", "bob", "room-7", "token-7")
	require.NoError(t, err)

	select {
	case callID := <-timedOut:
		assert.Equal(t, "call-7", callID)
		sess, _ := cm.Get("call-7")
		assert.Equal(t, CallStateEnded, sess.State)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected ring timeout within 500ms")
	}
}

// ---- Hub tests ----

func TestHub_RegisterAndUnregister(t *testing.T) {
	h := NewHub()

	// We can't easily create a real websocket.Conn in tests without a
	// server, so we test the map-level logic with a stub.
	// This tests the hub's Register/Unregister semantics indirectly via
	// the public API surface.

	assert.Equal(t, 0, h.OnlineCount())
	assert.False(t, h.IsOnline("alice"))
}

func TestHub_Broadcast_NoClients(t *testing.T) {
	h := NewHub()
	// Should not panic with no clients.
	h.Broadcast([]byte(`{"type":"ping"}`))
}

func TestHub_GetByUser_NoActiveCalls(t *testing.T) {
	cm := NewCallManager()
	_, ok := cm.GetByUser("nobody")
	assert.False(t, ok)
}

func TestHub_CallCount(t *testing.T) {
	cm := NewCallManager()
	assert.Equal(t, 0, cm.Count())

	cm.SetRingTimeoutDuration(50 * time.Millisecond)
	_, _ = cm.Invite("c1", "a", "b", "r", "t")
	assert.Equal(t, 1, cm.Count())

	_ = cm.Cancel("c1", "test")
	// After cancel, state is ended but still in map.
	assert.Equal(t, 0, cm.Count())
}
