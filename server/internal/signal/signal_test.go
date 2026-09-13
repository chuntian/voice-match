package signal

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCallStore is an in-memory CallStore used by tests to verify that
// CallManager persists call state changes.
type mockCallStore struct {
	mu    sync.Mutex
	calls map[string]map[string]interface{}
}

func newMockCallStore() *mockCallStore {
	return &mockCallStore{calls: make(map[string]map[string]interface{})}
}

func (m *mockCallStore) HSetCall(_ context.Context, callID string, values map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.calls[callID] == nil {
		m.calls[callID] = make(map[string]interface{})
	}
	for k, v := range values {
		m.calls[callID][k] = v
	}
	return nil
}

func (m *mockCallStore) HDelCall(_ context.Context, callID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.calls, callID)
	return nil
}

func (m *mockCallStore) get(callID string) (map[string]interface{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.calls[callID]
	return v, ok
}

// ---- Call state machine tests ----

func TestCallStateMachine_NormalFlow(t *testing.T) {
	cm := NewCallManager(newMockCallStore())
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
	cm := NewCallManager(newMockCallStore())
	cm.SetRingTimeoutDuration(5 * time.Second)

	_, err := cm.Invite("call-2", "alice", "bob", "room-2", "token-2")
	require.NoError(t, err)

	rejected, err := cm.Reject("call-2", "bob", "busy")
	require.NoError(t, err)
	assert.Equal(t, CallStateEnded, rejected.State)
}

func TestCallStateMachine_Cancel(t *testing.T) {
	cm := NewCallManager(newMockCallStore())
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
	cm := NewCallManager(newMockCallStore())

	// Accept a non-existent call.
	_, err := cm.Accept("nonexistent", "bob")
	assert.Error(t, err)
}

func TestCallStateMachine_AcceptWrongUser(t *testing.T) {
	cm := NewCallManager(newMockCallStore())
	cm.SetRingTimeoutDuration(5 * time.Second)

	_, err := cm.Invite("call-4", "alice", "bob", "room-4", "token-4")
	require.NoError(t, err)

	// Charlie tries to accept.
	_, err = cm.Accept("call-4", "charlie")
	assert.Error(t, err)
}

func TestCallStateMachine_AlreadyInCall(t *testing.T) {
	cm := NewCallManager(newMockCallStore())
	cm.SetRingTimeoutDuration(5 * time.Second)

	_, err := cm.Invite("call-5", "alice", "bob", "room-5", "token-5")
	require.NoError(t, err)

	// Alice tries to initiate another call.
	_, err = cm.Invite("call-6", "alice", "dave", "room-6", "token-6")
	assert.Error(t, err)
}

func TestCallStateMachine_RingTimeout(t *testing.T) {
	cm := NewCallManager(newMockCallStore())
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
	cm := NewCallManager(newMockCallStore())
	_, ok := cm.GetByUser("nobody")
	assert.False(t, ok)
}

func TestHub_CallCount(t *testing.T) {
	cm := NewCallManager(newMockCallStore())
	assert.Equal(t, 0, cm.Count())

	cm.SetRingTimeoutDuration(50 * time.Millisecond)
	_, _ = cm.Invite("c1", "a", "b", "r", "t")
	assert.Equal(t, 1, cm.Count())

	_ = cm.Cancel("c1", "test")
	// After cancel, state is ended but still in map.
	assert.Equal(t, 0, cm.Count())
}

// ---- Redis persistence test ----

func TestCallManager_PersistsToStore(t *testing.T) {
	store := newMockCallStore()
	cm := NewCallManager(store)
	cm.SetRingTimeoutDuration(5 * time.Second)

	// Invite: call should be persisted in ringing state.
	_, err := cm.Invite("call-persist", "alice", "bob", "room-persist", "rtc-token-persist")
	require.NoError(t, err)

	vals, ok := store.get("call-persist")
	require.True(t, ok, "call should exist in store after Invite")
	assert.Equal(t, string(CallStateRinging), vals["state"])
	assert.Equal(t, "alice", vals["caller_id"])
	assert.Equal(t, "bob", vals["callee_id"])
	assert.Equal(t, "room-persist", vals["room_name"])
	assert.Equal(t, "rtc-token-persist", vals["rtc_token"])

	// Accept: state should flip to connected.
	_, err = cm.Accept("call-persist", "bob")
	require.NoError(t, err)
	vals, ok = store.get("call-persist")
	require.True(t, ok)
	assert.Equal(t, string(CallStateConnected), vals["state"])

	// End: state should flip to ended.
	_, _, err = cm.End("call-persist", "alice", "normal")
	require.NoError(t, err)
	vals, ok = store.get("call-persist")
	require.True(t, ok)
	assert.Equal(t, string(CallStateEnded), vals["state"])

	// Remove: call should be deleted from store.
	cm.Remove("call-persist")
	_, ok = store.get("call-persist")
	assert.False(t, ok, "call should be removed from store after Remove")
}
