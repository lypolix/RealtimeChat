# Этап сборки
FROM golang:1.24 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/realtime-chat/main.go

# Этап запуска
FROM alpine:latest

WORKDIR /app

# Копируем необходимые файлы из builder
COPY --from=builder /app/app .
COPY --from=builder /app/config ./config
COPY --from=builder /app/.env .  

# Устанавливаем зависимости для alpine
RUN apk add --no-cache ca-certificates

# Пользователь для безопасности (не root)
RUN adduser -D appuser
USER appuser

EXPOSE 8080

CMD ["./app"]