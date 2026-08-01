FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/chamie ./cmd/chamie && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/migrateup ./cmd/migrateup

FROM alpine:3.21

RUN apk --no-cache add ca-certificates && \
    adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/chamie /out/migrateup ./
COPY db/migrations ./db/migrations
COPY prompts ./prompts

USER appuser

EXPOSE 8080

CMD ["sh", "-c", "./migrateup && exec ./chamie"]
