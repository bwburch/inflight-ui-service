package sessions

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	SessionDuration = 24 * time.Hour
	SessionPrefix   = "session:"
)

// Session represents an authenticated session
type Session struct {
	SessionID      string    `json:"session_id"`
	UserID         int       `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

// Store handles session persistence in Redis
type Store struct {
	redis *redis.Client
}

// NewStore creates a new session store
func NewStore(redisClient *redis.Client) *Store {
	return &Store{redis: redisClient}
}

// Create creates a new session for a user
func (s *Store) Create(ctx context.Context, userID int) (*Session, error) {
	// Generate cryptographically random session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}

	now := time.Now()
	session := &Session{
		SessionID:      sessionID,
		UserID:         userID,
		CreatedAt:      now,
		ExpiresAt:      now.Add(SessionDuration),
		LastActivityAt: now,
	}

	// Store in Redis with TTL
	key := SessionPrefix + sessionID
	err = s.redis.Set(ctx, key, userID, SessionDuration).Err()
	if err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}

	return session, nil
}

// Get retrieves a session by ID
func (s *Store) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := SessionPrefix + sessionID

	// Get user ID from Redis
	userID, err := s.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		return nil, nil // Session not found
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Get TTL to calculate expiration
	ttl, err := s.redis.TTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get session TTL: %w", err)
	}

	now := time.Now()
	session := &Session{
		SessionID:      sessionID,
		UserID:         userID,
		CreatedAt:      now.Add(-SessionDuration + ttl), // Approximate
		ExpiresAt:      now.Add(ttl),
		LastActivityAt: now,
	}

	return session, nil
}

// UpdateActivity extends the session TTL (sliding window)
func (s *Store) UpdateActivity(ctx context.Context, sessionID string) error {
	key := SessionPrefix + sessionID

	// Refresh TTL
	err := s.redis.Expire(ctx, key, SessionDuration).Err()
	if err != nil {
		return fmt.Errorf("update session activity: %w", err)
	}

	return nil
}

// Delete removes a session (logout)
func (s *Store) Delete(ctx context.Context, sessionID string) error {
	key := SessionPrefix + sessionID

	err := s.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

// DeleteAllUserSessions logs out all sessions for a user
func (s *Store) DeleteAllUserSessions(ctx context.Context, userID int) error {
	// Scan all session keys
	iter := s.redis.Scan(ctx, 0, SessionPrefix+"*", 0).Iterator()
	deleted := 0

	for iter.Next(ctx) {
		key := iter.Val()

		// Check if this session belongs to the user
		id, err := s.redis.Get(ctx, key).Int()
		if err == nil && id == userID {
			s.redis.Del(ctx, key)
			deleted++
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan sessions: %w", err)
	}

	return nil
}

// generateSessionID generates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32) // 256 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
