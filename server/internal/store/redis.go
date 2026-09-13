// Package store provides storage abstractions for the voice-match server.
//
// It defines interfaces for online state, call records, matching pools,
// and blacklists, with a Redis implementation for ephemeral data and a
// MySQL implementation for persistent records.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ---- Key prefix constants ----

const (
	keyOnlinePrefix    = "online:"
	keyCallPrefix      = "call:"
	keyPoolRandom      = "match:pool:random"
	keyPoolCityPrefix  = "match:pool:city:"
	keyPoolDestPrefix  = "match:pool:dest:"
	keyBlacklistPrefix = "user:blacklist:"
	keyPoolTimeoutZSet = "match:pool:timeout"
	keyUserPoolPrefix  = "match:userpool:"
)

// ---- Interfaces ----

// OnlineStore manages online presence.
type OnlineStore interface {
	SetOnline(ctx context.Context, userID, nodeInfo string, ttl time.Duration) error
	IsOnline(ctx context.Context, userID string) (bool, error)
	GetOnline(ctx context.Context, userID string) (string, error)
	DelOnline(ctx context.Context, userID string) error
}

// CallStore manages call state in Redis (for distributed coordination).
type CallStore interface {
	HSetCall(ctx context.Context, callID string, values map[string]interface{}) error
	HGetCall(ctx context.Context, callID, field string) (string, error)
	HDelCall(ctx context.Context, callID string) error
	CallExists(ctx context.Context, callID string) (bool, error)
}

// PoolStore manages matching pool lists.
type PoolStore interface {
	LPushPool(ctx context.Context, poolKey, userID string) error
	LPopPool(ctx context.Context, poolKey string) (string, error)
	LRemPool(ctx context.Context, poolKey, userID string) (int64, error)
	LLenPool(ctx context.Context, poolKey string) (int64, error)

	ZAddPoolTimeout(ctx context.Context, userID string, expireAt int64) error
	ZRemPoolTimeout(ctx context.Context, userID string) error
	ZRangeByScoreTimeout(ctx context.Context, maxScore int64) ([]string, error)
}

// BlacklistStore manages user blacklists.
type BlacklistStore interface {
	SAddBlacklist(ctx context.Context, userID, blockedUserID string) error
	SRemBlacklist(ctx context.Context, userID, blockedUserID string) error
	SIsBlacklist(ctx context.Context, userID, blockedUserID string) (bool, error)
	SMembersBlacklist(ctx context.Context, userID string) ([]string, error)
}

// RedisStore implements all four interfaces using go-redis v9.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a RedisStore from an existing go-redis client.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// NewRedisClient creates a new go-redis client with the given address.
func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// ---- OnlineStore implementation ----

func (r *RedisStore) SetOnline(ctx context.Context, userID, nodeInfo string, ttl time.Duration) error {
	key := keyOnlinePrefix + userID
	if err := r.client.Set(ctx, key, nodeInfo, ttl).Err(); err != nil {
		return fmt.Errorf("set online: %w", err)
	}
	return nil
}

func (r *RedisStore) IsOnline(ctx context.Context, userID string) (bool, error) {
	key := keyOnlinePrefix + userID
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check online: %w", err)
	}
	return n > 0, nil
}

func (r *RedisStore) GetOnline(ctx context.Context, userID string) (string, error) {
	key := keyOnlinePrefix + userID
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get online: %w", err)
	}
	return val, nil
}

func (r *RedisStore) DelOnline(ctx context.Context, userID string) error {
	key := keyOnlinePrefix + userID
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("del online: %w", err)
	}
	return nil
}

// ---- CallStore implementation ----

func (r *RedisStore) HSetCall(ctx context.Context, callID string, values map[string]interface{}) error {
	key := keyCallPrefix + callID
	if err := r.client.HSet(ctx, key, values).Err(); err != nil {
		return fmt.Errorf("hset call: %w", err)
	}
	// Set a TTL so orphaned calls don't accumulate.
	if err := r.client.Expire(ctx, key, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("expire call: %w", err)
	}
	return nil
}

func (r *RedisStore) HGetCall(ctx context.Context, callID, field string) (string, error) {
	key := keyCallPrefix + callID
	val, err := r.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("hget call: %w", err)
	}
	return val, nil
}

func (r *RedisStore) HDelCall(ctx context.Context, callID string) error {
	key := keyCallPrefix + callID
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("hdel call: %w", err)
	}
	return nil
}

func (r *RedisStore) CallExists(ctx context.Context, callID string) (bool, error) {
	key := keyCallPrefix + callID
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("call exists: %w", err)
	}
	return n > 0, nil
}

// ---- PoolStore implementation ----

func (r *RedisStore) LPushPool(ctx context.Context, poolKey, userID string) error {
	if err := r.client.LPush(ctx, poolKey, userID).Err(); err != nil {
		return fmt.Errorf("lpush pool: %w", err)
	}
	return nil
}

func (r *RedisStore) LPopPool(ctx context.Context, poolKey string) (string, error) {
	val, err := r.client.LPop(ctx, poolKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("lpop pool: %w", err)
	}
	return val, nil
}

func (r *RedisStore) LRemPool(ctx context.Context, poolKey, userID string) (int64, error) {
	n, err := r.client.LRem(ctx, poolKey, 0, userID).Result()
	if err != nil {
		return 0, fmt.Errorf("lrem pool: %w", err)
	}
	return n, nil
}

func (r *RedisStore) LLenPool(ctx context.Context, poolKey string) (int64, error) {
	n, err := r.client.LLen(ctx, poolKey).Result()
	if err != nil {
		return 0, fmt.Errorf("llen pool: %w", err)
	}
	return n, nil
}

func (r *RedisStore) ZAddPoolTimeout(ctx context.Context, userID string, expireAt int64) error {
	member := redis.Z{
		Score:  float64(expireAt),
		Member: userID,
	}
	if err := r.client.ZAdd(ctx, keyPoolTimeoutZSet, member).Err(); err != nil {
		return fmt.Errorf("zadd timeout: %w", err)
	}
	return nil
}

func (r *RedisStore) ZRemPoolTimeout(ctx context.Context, userID string) error {
	if err := r.client.ZRem(ctx, keyPoolTimeoutZSet, userID).Err(); err != nil {
		return fmt.Errorf("zrem timeout: %w", err)
	}
	return nil
}

func (r *RedisStore) ZRangeByScoreTimeout(ctx context.Context, maxScore int64) ([]string, error) {
	opt := redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", maxScore),
	}
	vals, err := r.client.ZRangeByScore(ctx, keyPoolTimeoutZSet, &opt).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange timeout: %w", err)
	}
	return vals, nil
}

// ListPools returns all active pool keys matching match:pool:*
func (r *RedisStore) ListPools(ctx context.Context) ([]string, error) {
	keys, err := r.client.Keys(ctx, "match:pool:*").Result()
	if err != nil {
		return nil, fmt.Errorf("list pools: %w", err)
	}
	return keys, nil
}

// SetUserPool records which pool a user joined as a hash field under
// match:userpool:{userID}, with a 10-minute TTL to avoid orphan records.
func (r *RedisStore) SetUserPool(ctx context.Context, userID, poolKey string) error {
	key := keyUserPoolPrefix + userID
	if err := r.client.HSet(ctx, key, "pool", poolKey).Err(); err != nil {
		return fmt.Errorf("hset user pool: %w", err)
	}
	if err := r.client.Expire(ctx, key, 10*time.Minute).Err(); err != nil {
		return fmt.Errorf("expire user pool: %w", err)
	}
	return nil
}

// GetUserPool returns the pool key a user is currently in, "" if none.
func (r *RedisStore) GetUserPool(ctx context.Context, userID string) (string, error) {
	key := keyUserPoolPrefix + userID
	val, err := r.client.HGet(ctx, key, "pool").Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("hget user pool: %w", err)
	}
	return val, nil
}

// DelUserPool removes the user->pool mapping.
func (r *RedisStore) DelUserPool(ctx context.Context, userID string) error {
	key := keyUserPoolPrefix + userID
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("del user pool: %w", err)
	}
	return nil
}

// ---- BlacklistStore implementation ----

func (r *RedisStore) SAddBlacklist(ctx context.Context, userID, blockedUserID string) error {
	key := keyBlacklistPrefix + userID
	if err := r.client.SAdd(ctx, key, blockedUserID).Err(); err != nil {
		return fmt.Errorf("sadd blacklist: %w", err)
	}
	return nil
}

func (r *RedisStore) SRemBlacklist(ctx context.Context, userID, blockedUserID string) error {
	key := keyBlacklistPrefix + userID
	if err := r.client.SRem(ctx, key, blockedUserID).Err(); err != nil {
		return fmt.Errorf("srem blacklist: %w", err)
	}
	return nil
}

func (r *RedisStore) SIsBlacklist(ctx context.Context, userID, blockedUserID string) (bool, error) {
	key := keyBlacklistPrefix + userID
	n, err := r.client.SIsMember(ctx, key, blockedUserID).Result()
	if err != nil {
		return false, fmt.Errorf("sisblacklist: %w", err)
	}
	return n, nil
}

func (r *RedisStore) SMembersBlacklist(ctx context.Context, userID string) ([]string, error) {
	key := keyBlacklistPrefix + userID
	vals, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("smembers blacklist: %w", err)
	}
	return vals, nil
}
