package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("key not found")

type Cacher interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	SAdd(ctx context.Context, key string, members ...string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SRem(ctx context.Context, key string, members ...string) error
}

type Redis struct {
	client *redis.Client
}

func NewRedis(url string) *Redis {
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic("invalid redis url: " + err.Error())
	}
	return &Redis{client: redis.NewClient(opt)}
}

func (r *Redis) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	return val, err
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *Redis) SAdd(ctx context.Context, key string, members ...string) error {
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	return r.client.SAdd(ctx, key, args...).Err()
}

func (r *Redis) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

func (r *Redis) SRem(ctx context.Context, key string, members ...string) error {
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	return r.client.SRem(ctx, key, args...).Err()
}
