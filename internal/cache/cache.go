package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	client *redis.Client
}

type SessionStoreInterface interface {
	SaveToken(ctx context.Context, token string, userID int, ttl time.Duration) error
	GetUserID(ctx context.Context, token string) (int, error)
	DeleteToken(ctx context.Context, token string) error
}

type FileCache interface {
	SetFileMetadata(ctx context.Context, userID int, docID string, meta string, ttl time.Duration) error
	GetFileMetadata(ctx context.Context, userID int, docID string) (string, error)
	InvalidateFile(ctx context.Context, userID int, docID string) error
	SetUserFiles(ctx context.Context, userID int, data string, ttl time.Duration) error
	GetUserFiles(ctx context.Context, userID int) (string, error)
	InvalidateUserFiles(ctx context.Context, userID int) error
	SetFileBytes(ctx context.Context, userID int, docID string, data []byte, ttl time.Duration) error
	GetFileBytes(ctx context.Context, userID int, docID string) ([]byte, error)
}

type RedisFileCache struct {
	client *redis.Client
}

func NewFileCache(addr, password string, db int) *RedisFileCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisFileCache{client: rdb}
}

func (c *RedisFileCache) fileKey(userID int, docID string) string {
	return fmt.Sprintf("file:%d:%s", userID, docID)
}

func (c *RedisFileCache) userFilesKey(userID int) string {
	return fmt.Sprintf("user:%d:files", userID)
}

func (c *RedisFileCache) fileBytesKey(userID int, docID string) string {
	return fmt.Sprintf("filebytes:%d:%s", userID, docID)
}

func (c *RedisFileCache) SetFileMetadata(ctx context.Context, userID int, docID string, meta string, ttl time.Duration) error {
	return c.client.Set(ctx, c.fileKey(userID, docID), meta, ttl).Err()
}

func (c *RedisFileCache) GetFileMetadata(ctx context.Context, userID int, docID string) (string, error) {
	return c.client.Get(ctx, c.fileKey(userID, docID)).Result()
}

func (c *RedisFileCache) InvalidateFile(ctx context.Context, userID int, docID string) error {
	return c.client.Del(ctx, c.fileKey(userID, docID)).Err()
}

func (c *RedisFileCache) SetUserFiles(ctx context.Context, userID int, data string, ttl time.Duration) error {
	return c.client.Set(ctx, c.userFilesKey(userID), data, ttl).Err()
}

func (c *RedisFileCache) GetUserFiles(ctx context.Context, userID int) (string, error) {
	return c.client.Get(ctx, c.userFilesKey(userID)).Result()
}

func (c *RedisFileCache) InvalidateUserFiles(ctx context.Context, userID int) error {
	return c.client.Del(ctx, c.userFilesKey(userID)).Err()
}

func (c *RedisFileCache) SetFileBytes(ctx context.Context, userID int, docID string, data []byte, ttl time.Duration) error {
	return c.client.Set(ctx, c.fileBytesKey(userID, docID), data, ttl).Err()
}

func (c *RedisFileCache) GetFileBytes(ctx context.Context, userID int, docID string) ([]byte, error) {
	return c.client.Get(ctx, c.fileBytesKey(userID, docID)).Bytes()
}

func NewSessionStore(addr, password string, db int) *SessionStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &SessionStore{client: rdb}
}

func (s *SessionStore) SaveToken(ctx context.Context, token string, userID int, ttl time.Duration) error {
	return s.client.Set(ctx, s.tokenKey(token), userID, ttl).Err()
}

func (s *SessionStore) GetUserID(ctx context.Context, token string) (int, error) {
	val, err := s.client.Get(ctx, s.tokenKey(token)).Result()
	if err != nil {
		return 0, err
	}
	var userID int

	_, err = fmt.Sscanf(val, "%d", &userID)
	if err != nil {
		return 0, fmt.Errorf("ошибка преобразования userID: %w", err)
	}
	return userID, nil
}

func (s *SessionStore) DeleteToken(ctx context.Context, token string) error {
	return s.client.Del(ctx, s.tokenKey(token)).Err()
}

func (s *SessionStore) tokenKey(token string) string {
	return fmt.Sprintf("auth:token:%s", token)
}
