package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/rueidis"
	"github.com/shopspring/decimal"
)

// Client wraps Redis operations using rueidis.
type Client struct {
	redis rueidis.Client
}

// NewClient creates a new Redis client.
func NewClient(ctx context.Context, url string) (*Client, error) {
	// Parse Redis URL (redis://localhost:6380)
	opts, err := rueidis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}

	client, err := rueidis.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf("create redis client: %w", err)
	}

	// Verify connection
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{redis: client}, nil
}

// Close closes the Redis client.
func (c *Client) Close() {
	c.redis.Close()
}

// Ping checks if Redis is reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.redis.Do(ctx, c.redis.B().Ping().Build()).Error()
}

// --- FX Rate Lock ---

// FXRateLock represents a locked FX rate for a quote.
type FXRateLock struct {
	QuoteID      string
	FromCurrency string
	ToCurrency   string
	Rate         decimal.Decimal
	ExpiresAt    time.Time
}

// LockFXRate locks an FX rate for a quote with TTL.
func (c *Client) LockFXRate(ctx context.Context, lock FXRateLock, ttl time.Duration) error {
	key := fmt.Sprintf("fx_rate:%s", lock.QuoteID)
	value := fmt.Sprintf("%s:%s:%s", lock.FromCurrency, lock.ToCurrency, lock.Rate.String())

	// Use SETNX with separate EXPIRE for atomic set-if-not-exists
	cmd := c.redis.B().Setnx().Key(key).Value(value).Build()
	set, err := c.redis.Do(ctx, cmd).AsBool()
	if err != nil {
		return fmt.Errorf("lock FX rate: %w", err)
	}
	if !set {
		return fmt.Errorf("rate already locked for quote %s", lock.QuoteID)
	}

	// Set expiration
	expireCmd := c.redis.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build()
	c.redis.Do(ctx, expireCmd)

	return nil
}

// GetFXRate retrieves a locked FX rate.
func (c *Client) GetFXRate(ctx context.Context, quoteID string) (*FXRateLock, error) {
	key := fmt.Sprintf("fx_rate:%s", quoteID)

	value, err := c.redis.Do(ctx, c.redis.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("get FX rate: %w", err)
	}

	// Parse value: "EUR:IDR:17500.50"
	var from, to, rateStr string
	_, err = fmt.Sscanf(value, "%3s:%3s:%s", &from, &to, &rateStr)
	if err != nil {
		return nil, fmt.Errorf("parse FX rate value: %w", err)
	}

	rate, err := decimal.NewFromString(rateStr)
	if err != nil {
		return nil, fmt.Errorf("parse rate: %w", err)
	}

	ttl, err := c.redis.Do(ctx, c.redis.B().Ttl().Key(key).Build()).ToInt64()
	if err != nil {
		ttl = 0
	}

	return &FXRateLock{
		QuoteID:      quoteID,
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         rate,
		ExpiresAt:    time.Now().Add(time.Duration(ttl) * time.Second),
	}, nil
}

// DeleteFXRate removes a locked FX rate.
func (c *Client) DeleteFXRate(ctx context.Context, quoteID string) error {
	key := fmt.Sprintf("fx_rate:%s", quoteID)
	return c.redis.Do(ctx, c.redis.B().Del().Key(key).Build()).Error()
}

// --- Rate Limiting ---

// CheckRateLimit checks if a tenant has exceeded their rate limit.
// Returns true if request is allowed, false if rate limited.
func (c *Client) CheckRateLimit(ctx context.Context, tenantID string, limitPerMinute int) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s", tenantID)
	now := time.Now().Unix()
	windowStart := now - 60 // 1 minute window

	// Use a Lua script for atomic rate limiting
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])

		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current requests
		local count = redis.call('ZCARD', key)

		if count < limit then
			-- Add current request
			redis.call('ZADD', key, now, now .. ':' .. math.random())
			redis.call('EXPIRE', key, 60)
			return 1
		else
			return 0
		end
	`

	result, err := c.redis.Do(ctx,
		c.redis.B().Eval().Script(script).Numkeys(1).Key(key).Arg(
			fmt.Sprintf("%d", now),
			fmt.Sprintf("%d", windowStart),
			fmt.Sprintf("%d", limitPerMinute),
		).Build(),
	).ToInt64()

	if err != nil {
		return false, fmt.Errorf("check rate limit: %w", err)
	}

	return result == 1, nil
}

// --- Idempotency ---

// SetIdempotencyKey sets an idempotency key with result.
func (c *Client) SetIdempotencyKey(ctx context.Context, tenantID, key string, result []byte, ttl time.Duration) error {
	redisKey := fmt.Sprintf("idempotency:%s:%s", tenantID, key)

	// Use SETNX for atomic set-if-not-exists
	cmd := c.redis.B().Setnx().Key(redisKey).Value(string(result)).Build()
	set, err := c.redis.Do(ctx, cmd).AsBool()
	if err != nil {
		return err
	}
	if !set {
		return fmt.Errorf("idempotency key already exists")
	}

	// Set expiration
	expireCmd := c.redis.B().Expire().Key(redisKey).Seconds(int64(ttl.Seconds())).Build()
	return c.redis.Do(ctx, expireCmd).Error()
}

// GetIdempotencyKey retrieves an idempotency result.
func (c *Client) GetIdempotencyKey(ctx context.Context, tenantID, key string) ([]byte, error) {
	redisKey := fmt.Sprintf("idempotency:%s:%s", tenantID, key)
	result, err := c.redis.Do(ctx, c.redis.B().Get().Key(redisKey).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, err
	}
	return []byte(result), nil
}

// --- Session/Cache ---

// CacheTenantByAPIKey caches tenant ID lookup by API key.
func (c *Client) CacheTenantByAPIKey(ctx context.Context, apiKeyHash, tenantID string, ttl time.Duration) error {
	key := fmt.Sprintf("api_key:%s", apiKeyHash)
	return c.redis.Do(ctx,
		c.redis.B().Set().Key(key).Value(tenantID).Ex(ttl).Build(),
	).Error()
}

// GetTenantByAPIKey retrieves cached tenant ID.
func (c *Client) GetTenantByAPIKey(ctx context.Context, apiKeyHash string) (string, error) {
	key := fmt.Sprintf("api_key:%s", apiKeyHash)
	result, err := c.redis.Do(ctx, c.redis.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return "", nil
		}
		return "", err
	}
	return result, nil
}
