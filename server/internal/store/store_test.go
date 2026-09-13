package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Mock Redis store ----

type mockRedisStore struct {
	online    map[string]string
	calls     map[string]map[string]interface{}
	pools     map[string][]string
	timeouts  map[string]int64
	blacklist map[string]map[string]bool
}

func newMockRedisStore() *mockRedisStore {
	return &mockRedisStore{
		online:    make(map[string]string),
		calls:     make(map[string]map[string]interface{}),
		pools:     make(map[string][]string),
		timeouts:  make(map[string]int64),
		blacklist: make(map[string]map[string]bool),
	}
}

func (m *mockRedisStore) SetOnline(_ context.Context, userID, nodeInfo string, _ time.Duration) error {
	m.online[userID] = nodeInfo
	return nil
}

func (m *mockRedisStore) IsOnline(_ context.Context, userID string) (bool, error) {
	_, ok := m.online[userID]
	return ok, nil
}

func (m *mockRedisStore) GetOnline(_ context.Context, userID string) (string, error) {
	return m.online[userID], nil
}

func (m *mockRedisStore) DelOnline(_ context.Context, userID string) error {
	delete(m.online, userID)
	return nil
}

func (m *mockRedisStore) HSetCall(_ context.Context, callID string, values map[string]interface{}) error {
	if m.calls[callID] == nil {
		m.calls[callID] = make(map[string]interface{})
	}
	for k, v := range values {
		m.calls[callID][k] = v
	}
	return nil
}

func (m *mockRedisStore) HGetCall(_ context.Context, callID, field string) (string, error) {
	if h, ok := m.calls[callID]; ok {
		if v, ok := h[field]; ok {
			return toString(v), nil
		}
	}
	return "", nil
}

func (m *mockRedisStore) HDelCall(_ context.Context, callID string) error {
	delete(m.calls, callID)
	return nil
}

func (m *mockRedisStore) CallExists(_ context.Context, callID string) (bool, error) {
	_, ok := m.calls[callID]
	return ok, nil
}

func (m *mockRedisStore) LPushPool(_ context.Context, poolKey, userID string) error {
	m.pools[poolKey] = append(m.pools[poolKey], userID)
	return nil
}

func (m *mockRedisStore) LPopPool(_ context.Context, poolKey string) (string, error) {
	list := m.pools[poolKey]
	if len(list) == 0 {
		return "", nil
	}
	popped := list[0]
	m.pools[poolKey] = list[1:]
	return popped, nil
}

func (m *mockRedisStore) LRemPool(_ context.Context, poolKey, userID string) (int64, error) {
	list := m.pools[poolKey]
	removed := int64(0)
	newList := make([]string, 0, len(list))
	for _, u := range list {
		if u == userID {
			removed++
		} else {
			newList = append(newList, u)
		}
	}
	m.pools[poolKey] = newList
	return removed, nil
}

func (m *mockRedisStore) LLenPool(_ context.Context, poolKey string) (int64, error) {
	return int64(len(m.pools[poolKey])), nil
}

func (m *mockRedisStore) ZAddPoolTimeout(_ context.Context, userID string, expireAt int64) error {
	m.timeouts[userID] = expireAt
	return nil
}

func (m *mockRedisStore) ZRemPoolTimeout(_ context.Context, userID string) error {
	delete(m.timeouts, userID)
	return nil
}

func (m *mockRedisStore) ZRangeByScoreTimeout(_ context.Context, maxScore int64) ([]string, error) {
	var result []string
	for uID, exp := range m.timeouts {
		if exp <= maxScore {
			result = append(result, uID)
		}
	}
	return result, nil
}

func (m *mockRedisStore) SAddBlacklist(_ context.Context, userID, blockedUserID string) error {
	if m.blacklist[userID] == nil {
		m.blacklist[userID] = make(map[string]bool)
	}
	m.blacklist[userID][blockedUserID] = true
	return nil
}

func (m *mockRedisStore) SRemBlacklist(_ context.Context, userID, blockedUserID string) error {
	if set, ok := m.blacklist[userID]; ok {
		delete(set, blockedUserID)
	}
	return nil
}

func (m *mockRedisStore) SIsBlacklist(_ context.Context, userID, blockedUserID string) (bool, error) {
	if set, ok := m.blacklist[userID]; ok {
		return set[blockedUserID], nil
	}
	return false, nil
}

func (m *mockRedisStore) SMembersBlacklist(_ context.Context, userID string) ([]string, error) {
	var result []string
	if set, ok := m.blacklist[userID]; ok {
		for u := range set {
			result = append(result, u)
		}
	}
	return result, nil
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return time.Unix(int64(t), 0).Format(time.RFC3339)
	case int64:
		return time.Unix(t, 0).Format(time.RFC3339)
	case float64:
		return time.Unix(int64(t), 0).Format(time.RFC3339)
	default:
		return ""
	}
}

// ---- Tests ----

func TestMockOnlineStore(t *testing.T) {
	s := newMockRedisStore()
	ctx := context.Background()

	online, _ := s.IsOnline(ctx, "alice")
	assert.False(t, online)

	_ = s.SetOnline(ctx, "alice", "node-1", 60*time.Second)
	online, _ = s.IsOnline(ctx, "alice")
	assert.True(t, online)

	node, _ := s.GetOnline(ctx, "alice")
	assert.Equal(t, "node-1", node)

	_ = s.DelOnline(ctx, "alice")
	online, _ = s.IsOnline(ctx, "alice")
	assert.False(t, online)
}

func TestMockCallStore(t *testing.T) {
	s := newMockRedisStore()
	ctx := context.Background()

	exists, _ := s.CallExists(ctx, "c1")
	assert.False(t, exists)

	_ = s.HSetCall(ctx, "c1", map[string]interface{}{
		"state":  "ringing",
		"caller": "alice",
	})

	exists, _ = s.CallExists(ctx, "c1")
	assert.True(t, exists)

	state, _ := s.HGetCall(ctx, "c1", "state")
	assert.Equal(t, "ringing", state)

	_ = s.HDelCall(ctx, "c1")
	exists, _ = s.CallExists(ctx, "c1")
	assert.False(t, exists)
}

func TestMockPoolStore(t *testing.T) {
	s := newMockRedisStore()
	ctx := context.Background()

	_ = s.LPushPool(ctx, "match:pool:random", "alice")
	_ = s.LPushPool(ctx, "match:pool:random", "bob")

	len, _ := s.LLenPool(ctx, "match:pool:random")
	assert.Equal(t, int64(2), len)

	popped, _ := s.LPopPool(ctx, "match:pool:random")
	assert.Equal(t, "alice", popped)

	removed, _ := s.LRemPool(ctx, "match:pool:random", "bob")
	assert.Equal(t, int64(1), removed)
}

func TestMockBlacklistStore(t *testing.T) {
	s := newMockRedisStore()
	ctx := context.Background()

	blocked, _ := s.SIsBlacklist(ctx, "alice", "bob")
	assert.False(t, blocked)

	_ = s.SAddBlacklist(ctx, "alice", "bob")
	blocked, _ = s.SIsBlacklist(ctx, "alice", "bob")
	assert.True(t, blocked)

	members, _ := s.SMembersBlacklist(ctx, "alice")
	assert.Contains(t, members, "bob")

	_ = s.SRemBlacklist(ctx, "alice", "bob")
	blocked, _ = s.SIsBlacklist(ctx, "alice", "bob")
	assert.False(t, blocked)
}

func TestMockTimeoutSet(t *testing.T) {
	s := newMockRedisStore()
	ctx := context.Background()

	now := time.Now().Unix()
	_ = s.ZAddPoolTimeout(ctx, "alice", now-100)
	_ = s.ZAddPoolTimeout(ctx, "bob", now+1000)

	expired, _ := s.ZRangeByScoreTimeout(ctx, now)
	require.Contains(t, expired, "alice")
	assert.NotContains(t, expired, "bob")
}
