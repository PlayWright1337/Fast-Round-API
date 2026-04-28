# Fast Round API

Черновик небольшого Go API для фиксации событий матча: судья отправляет событие, сервис атомарно обновляет состояние в Redis и публикует обновление в Pub/Sub канал для фронтенда или соседних сервисов.

## 1. Структура проекта

```text
fast-round-api/
├── go.mod
├── main.go
├── models/
│   └── match.go
├── storage/
│   └── redis.go
├── handlers/
│   └── events.go
└── .env
```

Инициализация:

```bash
go mod init fast-round-api
go get github.com/gin-gonic/gin
go get github.com/redis/go-redis/v9
```

## 2. Содержимое файлов

### `models/match.go`

```go
package models

type EventType string
type Team string

const (
	EventGoal     EventType = "goal"
	EventWinRound EventType = "win_round"

	TeamA Team = "a"
	TeamB Team = "b"
)

// EventRequest - событие, которое присылает судья.
type EventRequest struct {
	MatchID string    `json:"match_id" binding:"required"`
	Team    Team      `json:"team" binding:"required"`
	Type    EventType `json:"type" binding:"required"`
}

// MatchState - текущее состояние матча в Redis.
type MatchState struct {
	MatchID string `json:"match_id"`
	ScoreA  int64  `json:"score_a"`
	ScoreB  int64  `json:"score_b"`
	RoundsA int64  `json:"rounds_a"`
	RoundsB int64  `json:"rounds_b"`
}
```

### `storage/redis.go`

Главная часть здесь - атомарное обновление состояния.

```go
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

func NewRedisStore(addr string) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// UpdateMatchEvent обновляет счет и возвращает актуальное состояние.
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
```

### `handlers/events.go`

```go
package handlers

import (
	"net/http"

	"fast-round-api/models"
	"fast-round-api/storage"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	Store *storage.RedisStore
}

func (h *EventHandler) HandleEvent(c *gin.Context) {
	var req models.EventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	newState, err := h.Store.UpdateMatchEvent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Store.PublishUpdate(c.Request.Context(), "match_updates", newState); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "publish update failed"})
		return
	}

	c.JSON(http.StatusOK, newState)
}

func (h *EventHandler) HandleGetMatch(c *gin.Context) {
	matchID := c.Param("match_id")

	state, err := h.Store.GetMatchState(c.Request.Context(), matchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get match failed"})
		return
	}

	c.JSON(http.StatusOK, state)
}
```

### `main.go`

```go
package main

import (
	"context"
	"log"
	"os"

	"fast-round-api/handlers"
	"fast-round-api/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	port := getenv("PORT", "8080")

	store := storage.NewRedisStore(redisAddr)
	if err := store.Ping(context.Background()); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	handler := &handlers.EventHandler{Store: store}

	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		v1.POST("/event", handler.HandleEvent)
		v1.GET("/matches/:match_id", handler.HandleGetMatch)
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
```

### `.env`

```env
PORT=8080
REDIS_ADDR=localhost:6379
```

## 3. Запуск

Redis локально:

```bash
docker run --name fast-round-redis -p 6379:6379 -d redis:7-alpine
```

Сервер:

```bash
go run ./...
```

## 4. Проверка

Засчитать гол команде A:

```bash
curl -X POST http://localhost:8080/api/v1/event \
  -H "Content-Type: application/json" \
  -d '{"match_id":"top_match_1","team":"a","type":"goal"}'
```

Завершить раунд в пользу команды B:

```bash
curl -X POST http://localhost:8080/api/v1/event \
  -H "Content-Type: application/json" \
  -d '{"match_id":"top_match_1","team":"b","type":"win_round"}'
```

Получить состояние матча:

```bash
curl http://localhost:8080/api/v1/matches/top_match_1
```

## 5. Что это дает

Каждое событие судьи быстро попадает в Redis, состояние матча обновляется атомарно, а другие сервисы могут слушать канал `match_updates` и сразу обновлять экран зрителя, оверлей трансляции или административную панель.

## 6. Что я бы добавил следующим шагом

1. **Идемпотентность событий.** Добавить `event_id`, чтобы повторный запрос от судейского клиента не засчитывал гол или раунд дважды.
2. **Историю событий.** Писать каждое событие в Redis Stream или обычную БД, чтобы можно было восстановить матч, показать таймлайн и разбирать спорные моменты.
3. **WebSocket/SSE endpoint.** Сейчас Pub/Sub удобен для внутренних сервисов, но фронтенду нужен отдельный канал: `/api/v1/matches/:match_id/stream`.
4. **Авторизацию судьи.** Хотя бы API key/JWT, иначе любой клиент сможет менять счет.
5. **Тесты.** Покрыть `UpdateMatchEvent`: гол, победа в раунде, сброс очков, неизвестная команда, неизвестный тип события.
6. **Docker Compose.** Поднять API и Redis одной командой, чтобы проект можно было запускать без ручной настройки.
7. **Правила матча.** Например, максимум очков в раунде, максимум раундов, запрет событий после завершения матча.
