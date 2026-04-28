package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"fast-round-api/models"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

type RedisOptions struct {
	Addr     string
	Password string
	DB       int
}

func NewRedisStore(options RedisOptions) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr:     options.Addr,
			Password: options.Password,
			DB:       options.DB,
		}),
	}
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) UpdateMatchEvent(ctx context.Context, req models.EventRequest) (*models.MatchState, error) {
	if req.Team != models.TeamA && req.Team != models.TeamB {
		return nil, errors.New("unknown team")
	}

	key := "match:" + req.MatchID
	teamSuffix := string(req.Team)

	pipe := s.client.TxPipeline()

	switch req.Type {
	case models.EventGoal:
		pipe.HIncrBy(ctx, key, "score_"+teamSuffix, 1)
	case models.EventWinRound:
		pipe.HIncrBy(ctx, key, "rounds_"+teamSuffix, 1)
		pipe.HSet(ctx, key, "score_a", 0, "score_b", 0)
	default:
		return nil, errors.New("unknown event type")
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	return s.GetMatchState(ctx, req.MatchID)
}

func (s *RedisStore) GetMatchState(ctx context.Context, matchID string) (*models.MatchState, error) {
	key := "match:" + matchID

	res, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return &models.MatchState{
		MatchID: matchID,
		ScoreA:  toInt64(res["score_a"]),
		ScoreB:  toInt64(res["score_b"]),
		RoundsA: toInt64(res["rounds_a"]),
		RoundsB: toInt64(res["rounds_b"]),
	}, nil
}

func (s *RedisStore) PublishUpdate(ctx context.Context, channel string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	return s.client.Publish(ctx, channel, payload).Err()
}

func toInt64(value string) int64 {
	result, _ := strconv.ParseInt(value, 10, 64)
	return result
}
