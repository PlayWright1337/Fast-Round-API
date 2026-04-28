GO ?= go
GOFMT ?= gofmt
APP ?= fast-round-api

.PHONY: tidy fmt test build run compose-up compose-down health example-goal example-round

tidy:
	$(GO) mod tidy

fmt:
	$(GOFMT) -w main.go config/*.go handlers/*.go models/*.go storage/*.go

test:
	$(GO) test ./...

build:
	$(GO) build -o $(APP) .

run:
	$(GO) run .

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

health:
	curl -f http://localhost:8080/health

example-goal:
	curl -X POST http://localhost:8080/api/v1/event -H "Content-Type: application/json" -H "X-API-Key: change-me" -d '{"match_id":"top_match_1","team":"a","type":"goal"}'

example-round:
	curl -X POST http://localhost:8080/api/v1/event -H "Content-Type: application/json" -H "X-API-Key: change-me" -d '{"match_id":"top_match_1","team":"b","type":"win_round"}'
