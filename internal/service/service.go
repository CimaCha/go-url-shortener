package service

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrEmptyURL    = errors.New("empty URL")
	ErrURLNotFound = errors.New("URL not found")
)

const urlTTL = 30 * 24 * time.Hour

type Service struct {
	redisClient *redis.Client
}

func NewService(redisClient *redis.Client) Service {
	return Service{redisClient: redisClient}
}

func ShortenUrl(fullURL string) string {
	digest := sha1.Sum([]byte(fullURL))
	shortURL := base64.RawURLEncoding.EncodeToString(digest[:])
	return shortURL
}

func (s Service) SetShortUrl(ctx context.Context, fullURL string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	shortUrl := ShortenUrl(fullURL)
	if err := s.redisClient.Set(ctx, shortUrl, fullURL, urlTTL).Err(); err != nil {
		return "", err
	}
	return shortUrl, nil
}

func (s Service) GetFullUrl(ctx context.Context, shortUrl string) (string, error) {
	if shortUrl == "" {
		return "", ErrEmptyURL
	}
	fullURL, err := s.redisClient.Get(ctx, shortUrl).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrURLNotFound
	}
	if err != nil {
		return "", err
	}
	return fullURL, nil
}
