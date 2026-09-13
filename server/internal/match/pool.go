// Package match implements the matching pool logic.
//
// Three pool types are supported:
//   - random:   global list match:pool:random
//   - city:     geohash-prefixed list match:pool:city:{geohash_prefix4}
//   - dest:     destination-tagged list match:pool:dest:{city_id}
//
// The Matcher runs a polling loop that LPOPs two users from each pool
// every 500ms, checks the blacklist, and emits match_found notifications.
package match

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PoolStore abstracts the Redis operations needed by the matcher.
type PoolStore interface {
	// LPushPool pushes a userID onto the named pool list.
	LPushPool(ctx context.Context, poolKey, userID string) error
	// LPopPool pops one userID from the left of the pool list.
	LPopPool(ctx context.Context, poolKey string) (string, error)
	// LRemPool removes all occurrences of userID from the pool list.
	LRemPool(ctx context.Context, poolKey, userID string) (int64, error)
	// LLenPool returns the length of the pool list.
	LLenPool(ctx context.Context, poolKey string) (int64, error)

	// ZAddPoolTimeout adds a userID to the timeout sorted set with score=unix timestamp.
	ZAddPoolTimeout(ctx context.Context, userID string, expireAt int64) error
	// ZRemPoolTimeout removes a userID from the timeout set.
	ZRemPoolTimeout(ctx context.Context, userID string) error
	// ZRangeByScoreTimeout returns userIDs whose expiration <= now.
	ZRangeByScoreTimeout(ctx context.Context, maxScore int64) ([]string, error)

	// SIsBlacklist checks if targetUser is in the blocker's blacklist set.
	SIsBlacklist(ctx context.Context, blockerID, targetUser string) (bool, error)

	// ListPools returns all active pool keys matching match:pool:*
	ListPools(ctx context.Context) ([]string, error)
	// SetUserPool records which pool a user joined (Redis hash match:userpool:{userID}, field "pool")
	SetUserPool(ctx context.Context, userID, poolKey string) error
	// GetUserPool returns the pool key a user is currently in, "" if none
	GetUserPool(ctx context.Context, userID string) (string, error)
	// DelUserPool removes the user->pool mapping
	DelUserPool(ctx context.Context, userID string) error
}

// NotificationSink is called when a match is found. The signal layer
// implements this to deliver match_found messages to both parties.
type NotificationSink interface {
	// OnMatchFound is invoked with both matched user IDs and the match type.
	OnMatchFound(userA, userB, matchType string)
}

// Matcher manages the matching pools.
type Matcher struct {
	store PoolStore
	sink  NotificationSink

	pollInterval   time.Duration
	timeoutScanInterval time.Duration
	poolTTL        time.Duration // how long a user stays in the pool

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewMatcher creates a Matcher with default settings.
func NewMatcher(store PoolStore, sink NotificationSink) *Matcher {
	return &Matcher{
		store:           store,
		sink:            sink,
		pollInterval:    500 * time.Millisecond,
		timeoutScanInterval: 10 * time.Second,
		poolTTL:         60 * time.Second,
	}
}

// SetPollInterval overrides the matching poll interval.
func (m *Matcher) SetPollInterval(d time.Duration) { m.pollInterval = d }

// SetPoolTTL overrides the pool entry TTL.
func (m *Matcher) SetPoolTTL(d time.Duration) { m.poolTTL = d }

// poolKeyFor returns the Redis key for the given match type and parameters.
func poolKeyFor(matchType, city, geohash, destination string) string {
	switch matchType {
	case "city":
		prefix := geohash
		if len(prefix) > 4 {
			prefix = prefix[:4]
		}
		return fmt.Sprintf("match:pool:city:%s", prefix)
	case "destination":
		return fmt.Sprintf("match:pool:dest:%s", destination)
	default:
		return "match:pool:random"
	}
}

// JoinPool adds a user to the matching pool.
func (m *Matcher) JoinPool(userID, matchType, city, geohash, destination string, tags []string, genderPreference string) (int, int, error) {
	ctx := context.Background()
	key := poolKeyFor(matchType, city, geohash, destination)

	if err := m.store.LPushPool(ctx, key, userID); err != nil {
		return 0, 0, fmt.Errorf("lpush pool: %w", err)
	}

	// Record which pool this user joined so we can later remove them
	// from the correct list (city/dest pools are discovered dynamically).
	if err := m.store.SetUserPool(ctx, userID, key); err != nil {
		log.Printf("[match] warn: record user pool failed for %s: %v", userID, err)
	}

	expireAt := time.Now().Add(m.poolTTL).Unix()
	if err := m.store.ZAddPoolTimeout(ctx, userID, expireAt); err != nil {
		log.Printf("[match] warn: set pool timeout failed for %s: %v", userID, err)
	}

	size, err := m.store.LLenPool(ctx, key)
	if err != nil {
		size = 0
	}
	waitTime := int(m.poolTTL.Seconds())
	return int(size), waitTime, nil
}

// LeavePool removes a user from all matching pools. The matchType argument
// is ignored: the actual pool is located via the user->pool mapping so that
// city/destination pools are removed correctly.
func (m *Matcher) LeavePool(userID, matchType string) error {
	ctx := context.Background()

	// Locate the pool the user actually joined via the user->pool mapping.
	if poolKey, err := m.store.GetUserPool(ctx, userID); err == nil && poolKey != "" {
		if _, lerr := m.store.LRemPool(ctx, poolKey, userID); lerr != nil {
			log.Printf("[match] warn: lrem user %s from %s failed: %v", userID, poolKey, lerr)
		}
		if derr := m.store.DelUserPool(ctx, userID); derr != nil {
			log.Printf("[match] warn: del user pool mapping for %s failed: %v", userID, derr)
		}
	} else if err != nil {
		log.Printf("[match] warn: get user pool for %s failed: %v", userID, err)
	}

	// Best-effort fallback: also remove from the random pool in case the
	// mapping was lost or never recorded.
	_, _ = m.store.LRemPool(ctx, "match:pool:random", userID)

	// Also remove from timeout set.
	_ = m.store.ZRemPoolTimeout(ctx, userID)

	return nil
}

// Start begins the polling loop. It returns immediately; the matcher
// runs in background goroutines.
func (m *Matcher) Start(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	ctx, m.cancel = context.WithCancel(ctx)
	m.running = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.pollLoop(ctx)

	m.wg.Add(1)
	go m.timeoutLoop(ctx)

	log.Printf("[match] matcher started")
}

// Stop halts the matcher.
func (m *Matcher) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	m.running = false
	log.Printf("[match] matcher stopped")
}

// pollLoop runs the matching poll.
func (m *Matcher) pollLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Discover all active pools dynamically (random, city:*, dest:*).
			poolKeys, err := m.store.ListPools(ctx)
			if err != nil {
				log.Printf("[match] list pools error: %v", err)
				continue
			}
			for _, poolKey := range poolKeys {
				m.matchOneFromPool(ctx, poolKey)
			}
		}
	}
}

// matchOneFromPool pops two users from the given pool, checks blacklist,
// and notifies.
func (m *Matcher) matchOneFromPool(ctx context.Context, poolKey string) {
	// Pop user A.
	userA, err := m.store.LPopPool(ctx, poolKey)
	if err != nil || userA == "" {
		return
	}

	// Pop user B.
	userB, err := m.store.LPopPool(ctx, poolKey)
	if err != nil || userB == "" {
		// Put A back at the front.
		_ = m.store.LPushPool(ctx, poolKey, userA)
		return
	}

	// Check blacklist both ways.
	blocked, err := m.store.SIsBlacklist(ctx, userA, userB)
	if err != nil {
		log.Printf("[match] blacklist check error: %v", err)
	}
	if blocked {
		// Put B back; skip A (they'll timeout).
		_ = m.store.LPushPool(ctx, poolKey, userB)
		return
	}
	blocked, err = m.store.SIsBlacklist(ctx, userB, userA)
	if err != nil {
		log.Printf("[match] blacklist check error: %v", err)
	}
	if blocked {
		_ = m.store.LPushPool(ctx, poolKey, userB)
		return
	}

	// Clear timeout tracking for both.
	_ = m.store.ZRemPoolTimeout(ctx, userA)
	_ = m.store.ZRemPoolTimeout(ctx, userB)

	// Emit match.
	matchType := "random"
	if len(poolKey) > len("match:pool:") {
		matchType = poolKey[len("match:pool:"):]
	}
	log.Printf("[match] paired users %s <-> %s in pool %s", userA, userB, poolKey)

	if m.sink != nil {
		m.sink.OnMatchFound(userA, userB, matchType)
	}
}

// timeoutLoop periodically scans for expired pool entries and evicts them.
func (m *Matcher) timeoutLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.timeoutScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Unix()
			expired, err := m.store.ZRangeByScoreTimeout(ctx, now)
			if err != nil {
				log.Printf("[match] timeout scan error: %v", err)
				continue
			}
			for _, userID := range expired {
				log.Printf("[match] pool timeout evict user=%s", userID)
				// Locate the pool the user was actually in via the
				// user->pool mapping, and remove them from it.
				if poolKey, gerr := m.store.GetUserPool(ctx, userID); gerr == nil && poolKey != "" {
					_, _ = m.store.LRemPool(ctx, poolKey, userID)
					_ = m.store.DelUserPool(ctx, userID)
				}
				// Best-effort fallback: also remove from the random pool.
				_, _ = m.store.LRemPool(ctx, "match:pool:random", userID)
				_ = m.store.ZRemPoolTimeout(ctx, userID)
			}
		}
	}
}
