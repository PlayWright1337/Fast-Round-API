FROM golang:1.26.2-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/fast-round-api .

FROM alpine:3.21

WORKDIR /app

RUN adduser -D -H appuser

COPY --from=build /out/fast-round-api /app/fast-round-api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/fast-round-api"]
