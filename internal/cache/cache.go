package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type FileCache interface {
	SetFileMetadata(ctx context.Context, userID int, docID string, meta string, ttl time.Duration) error
	InvalidateFile(ctx context.Context, userID int, docID string) error
	SetUserFiles(ctx context.Context, userID int, data string, ttl time.Duration) error
	GetUserFiles(ctx context.Context, userID int) (string, error)
	InvalidateUserFiles(ctx context.Context, userID int) error
	SetFileBytes(ctx context.Context, userID int, docID string, data []byte, ttl time.Duration) error
	GetFileBytes(ctx context.Context, userID int, docID string) ([]byte, error)
}

// Общий Redis-клиент
type RedisClient struct {
	client *redis.Client
}

// RedisFileCache реализация FileCache
type RedisFileCache struct {
	*RedisClient
}

// SessionStore для токенов
type SessionStore struct {
	*RedisClient
}

func NewRedisClient(addr, password string, db int) *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func NewSessionStore(addr, password string, db int) *SessionStore {
	return &SessionStore{NewRedisClient(addr, password, db)}
}

func (s *SessionStore) SaveToken(ctx context.Context, token string, userID int, ttl time.Duration) error {
	return s.client.Set(ctx, fmt.Sprintf("auth:token:%s", token), userID, ttl).Err()
}

func (s *SessionStore) GetUserID(ctx context.Context, token string) (int, error) {
	val, err := s.client.Get(ctx, fmt.Sprintf("auth:token:%s", token)).Result()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

func (s *SessionStore) DeleteToken(ctx context.Context, token string) error {
	return s.client.Del(ctx, fmt.Sprintf("auth:token:%s", token)).Err()
}

func NewFileCache(addr, password string, db int) FileCache {
	return &RedisFileCache{NewRedisClient(addr, password, db)}
}

func (c *RedisFileCache) SetFileMetadata(ctx context.Context, userID int, docID string, meta string, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf("file:%d:%s", userID, docID), meta, ttl).Err()
}

func (c *RedisFileCache) GetFileMetadata(ctx context.Context, userID int, docID string) (string, error) {
	return c.client.Get(ctx, fmt.Sprintf("file:%d:%s", userID, docID)).Result()
}

func (c *RedisFileCache) InvalidateFile(ctx context.Context, userID int, docID string) error {
	return c.client.Del(ctx, fmt.Sprintf("file:%d:%s", userID, docID)).Err()
}

func (c *RedisFileCache) SetUserFiles(ctx context.Context, userID int, data string, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf("user:%d:files", userID), data, ttl).Err()
}

func (c *RedisFileCache) GetUserFiles(ctx context.Context, userID int) (string, error) {
	return c.client.Get(ctx, fmt.Sprintf("user:%d:files", userID)).Result()
}

func (c *RedisFileCache) InvalidateUserFiles(ctx context.Context, userID int) error {
	return c.client.Del(ctx, fmt.Sprintf("user:%d:files", userID)).Err()
}

func (c *RedisFileCache) SetFileBytes(ctx context.Context, userID int, docID string, data []byte, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf("filebytes:%d:%s", userID, docID), data, ttl).Err()
}

func (c *RedisFileCache) GetFileBytes(ctx context.Context, userID int, docID string) ([]byte, error) {
	return c.client.Get(ctx, fmt.Sprintf("filebytes:%d:%s", userID, docID)).Bytes()
}
