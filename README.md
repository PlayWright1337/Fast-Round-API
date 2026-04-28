# Fast Round API

HTTP API на Go для быстрого учета событий матча. Судья отправляет событие, сервис обновляет состояние в Redis и публикует новое состояние в канал `match_updates`.

## Быстрый запуск

```bash
cp .env.example .env
sed -i 's/API_KEY=change-me/API_KEY=your-secret-key/' .env
docker compose up --build -d
curl http://localhost:8080/health
```

## Запуск без Docker для API

Нужны Go 1.26+ и Redis.

```bash
cp .env.example .env
docker run --name fast-round-redis -p 127.0.0.1:6379:6379 -d redis:7-alpine
go run .
```

## Настройки

| Переменная | Значение по умолчанию | Описание |
| --- | --- | --- |
| `PORT` | `8080` | Порт HTTP сервера |
| `GIN_MODE` | `release` | Режим Gin |
| `REDIS_ADDR` | `localhost:6379` | Адрес Redis |
| `REDIS_PASSWORD` | пусто | Пароль Redis |
| `REDIS_DB` | `0` | Номер базы Redis |
| `API_KEY` | `change-me` в `.env.example` | Если задан, `POST /api/v1/event` требует заголовок `X-API-Key` |
| `MAX_BODY_BYTES` | `1048576` | Максимальный размер HTTP тела |
| `READ_TIMEOUT` | `5s` | Таймаут чтения HTTP запроса |
| `WRITE_TIMEOUT` | `5s` | Таймаут ответа HTTP сервера |
| `IDLE_TIMEOUT` | `60s` | Таймаут keep-alive соединения |
| `SHUTDOWN_TIMEOUT` | `10s` | Таймаут graceful shutdown |

## Безопасность

Минимум для нормальной установки:

```bash
cp .env.example .env
sed -i 's/API_KEY=change-me/API_KEY='$(openssl rand -hex 32)'/' .env
```

Что уже включено:

| Механизм | Где |
| --- | --- |
| API key для записи событий | `X-API-Key` на `POST /api/v1/event` |
| Redis не публикуется наружу в Docker Compose | `expose`, без `ports` |
| Лимит тела запроса | `MAX_BODY_BYTES` |
| HTTP timeouts | `READ_TIMEOUT`, `WRITE_TIMEOUT`, `IDLE_TIMEOUT` |
| Security headers | `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy` |
| Отключены доверенные proxy по умолчанию | `SetTrustedProxies(nil)` |

Не делай так на сервере:

```bash
docker run --name fast-round-redis -p 6379:6379 -d redis:7-alpine
```

Если Redis нужен локально без Compose, привязывай его только к localhost:

```bash
docker run --name fast-round-redis -p 127.0.0.1:6379:6379 -d redis:7-alpine
```

## API

### Health

```bash
curl http://localhost:8080/health
```

### Добавить событие

```bash
curl -X POST http://localhost:8080/api/v1/event \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-secret-key" \
  -d '{"match_id":"top_match_1","team":"a","type":"goal"}'
```

Если `API_KEY` пустой, заголовок не нужен. Для реальной установки оставлять `API_KEY` пустым не стоит.

Типы событий:

| type | Что делает |
| --- | --- |
| `goal` | Добавляет 1 очко команде |
| `win_round` | Добавляет 1 выигранный раунд команде и сбрасывает очки |

Команды:

| team | Команда |
| --- | --- |
| `a` | Команда A |
| `b` | Команда B |

### Получить состояние матча

```bash
curl http://localhost:8080/api/v1/matches/top_match_1
```

Ответ:

```json
{
  "match_id": "top_match_1",
  "score_a": 1,
  "score_b": 0,
  "rounds_a": 0,
  "rounds_b": 0
}
```

## Команды

```bash
make test
make build
make run
make compose-up
make compose-down
make health
make example-goal
make example-round
```

## Установка бинарника

```bash
go build -o fast-round-api .
PORT=8080 REDIS_ADDR=localhost:6379 ./fast-round-api
```

## Redis Pub/Sub

После каждого успешного события сервис публикует JSON состояния матча в канал:

```text
match_updates
```
