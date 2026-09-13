package match

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Mock store ----

type mockPoolStore struct {
	mu          sync.Mutex
	pools       map[string][]string           // poolKey -> list
	timeouts    map[string]int64               // userID -> expireAt (score)
	blacklists  map[string]map[string]bool     // blocker -> set of targets
	lpopErrors  map[string]error
	lpushErrors map[string]error
}

func newMockPoolStore() *mockPoolStore {
	return &mockPoolStore{
		pools:      make(map[string][]string),
		timeouts:   make(map[string]int64),
		blacklists: make(map[string]map[string]bool),
	}
}

func (m *mockPoolStore) LPushPool(_ context.Context, poolKey, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.lpushErrors[poolKey]; ok {
		return err
	}
	m.pools[poolKey] = append(m.pools[poolKey], userID)
	return nil
}

func (m *mockPoolStore) LPopPool(_ context.Context, poolKey string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.lpopErrors[poolKey]; ok {
		return "", err
	}
	list := m.pools[poolKey]
	if len(list) == 0 {
		return "", nil
	}
	popped := list[0]
	m.pools[poolKey] = list[1:]
	return popped, nil
}

func (m *mockPoolStore) LRemPool(_ context.Context, poolKey, userID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.pools[poolKey]
	var removed int64
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

func (m *mockPoolStore) LLenPool(_ context.Context, poolKey string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.pools[poolKey])), nil
}

func (m *mockPoolStore) ZAddPoolTimeout(_ context.Context, userID string, expireAt int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeouts[userID] = expireAt
	return nil
}

func (m *mockPoolStore) ZRemPoolTimeout(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.timeouts, userID)
	return nil
}

func (m *mockPoolStore) ZRangeByScoreTimeout(_ context.Context, maxScore int64) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []string
	for uID, exp := range m.timeouts {
		if exp <= maxScore {
			result = append(result, uID)
		}
	}
	return result, nil
}

func (m *mockPoolStore) SIsBlacklist(_ context.Context, blockerID, targetUser string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if set, ok := m.blacklists[blockerID]; ok {
		return set[targetUser], nil
	}
	return false, nil
}

// ---- Mock sink ----

type mockSink struct {
	mu    sync.Mutex
	matches []matchResult
}

type matchResult struct {
	userA, userB, matchType string
}

func (s *mockSink) OnMatchFound(userA, userB, matchType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matches = append(s.matches, matchResult{userA, userB, matchType})
}

func (s *mockSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.matches)
}

// ---- Tests ----

func TestMatcher_JoinPool(t *testing.T) {
	store := newMockPoolStore()
	sink := &mockSink{}
	m := NewMatcher(store, sink)

	size, wait, err := m.JoinPool("alice", "random", "", "", "", nil, "any")
	require.NoError(t, err)
	assert.Equal(t, 1, size)
	assert.Equal(t, 60, wait)

	// Verify in store.
	store.mu.Lock()
	list := store.pools["match:pool:random"]
	store.mu.Unlock()
	assert.Contains(t, list, "alice")
}

func TestMatcher_LeavePool(t *testing.T) {
	store := newMockPoolStore()
	sink := &mockSink{}
	m := NewMatcher(store, sink)

	_, _, _ = m.JoinPool("alice", "random", "", "", "", nil, "any")
	_, _, _ = m.JoinPool("bob", "random", "", "", "", nil, "any")

	err := m.LeavePool("alice", "random")
	require.NoError(t, err)

	store.mu.Lock()
	list := store.pools["match:pool:random"]
	store.mu.Unlock()
	assert.NotContains(t, list, "alice")
	assert.Contains(t, list, "bob")
}

func TestMatcher_PairTwoUsers(t *testing.T) {
	store := newMockPoolStore()
	sink := &mockSink{}
	m := NewMatcher(store, sink)
	m.SetPollInterval(50 * time.Millisecond)

	_, _, _ = m.JoinPool("alice", "random", "", "", "", nil, "any")
	_, _, _ = m.JoinPool("bob", "random", "", "", "", nil, "any")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	// Wait for match.
	require.Eventually(t, func() bool {
		return sink.count() >= 1
	}, 500*time.Millisecond, 20*time.Millisecond)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	assert.Equal(t, "alice", sink.matches[0].userA)
	assert.Equal(t, "bob", sink.matches[0].userB)
}

func TestMatcher_BlacklistSkips(t *testing.T) {
	store := newMockPoolStore()
	// alice has blacklisted bob.
	store.blacklists["alice"] = map[string]bool{"bob": true}

	sink := &mockSink{}
	m := NewMatcher(store, sink)
	m.SetPollInterval(50 * time.Millisecond)

	_, _, _ = m.JoinPool("alice", "random", "", "", "", nil, "any")
	_, _, _ = m.JoinPool("bob", "random", "", "", "", nil, "any")
	_, _, _ = m.JoinPool("charlie", "random", "", "", "", nil, "any")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	// After a while, bob should be pushed back and charlie matched with alice.
	require.Eventually(t, func() bool {
		return sink.count() >= 1
	}, 800*time.Millisecond, 30*time.Millisecond)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	// The match should NOT be alice <-> bob.
	for _, mr := range sink.matches {
		if (mr.userA == "alice" && mr.userB == "bob") ||
			(mr.userA == "bob" && mr.userB == "alice") {
			t.Fatalf("blacklisted pair should not match: %+v", mr)
		}
	}
}

func TestMatcher_PoolKeyFor_City(t *testing.T) {
	key := poolKeyFor("city", "hangzhou", "u12345678", "")
	assert.Equal(t, "match:pool:city:u123", key)
}

func TestMatcher_PoolKeyFor_Destination(t *testing.T) {
	key := poolKeyFor("destination", "", "", "hangzhou")
	assert.Equal(t, "match:pool:dest:hangzhou", key)
}

func TestMatcher_PoolKeyFor_Random(t *testing.T) {
	key := poolKeyFor("random", "", "", "")
	assert.Equal(t, "match:pool:random", key)
}
