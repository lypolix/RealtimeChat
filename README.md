# RealtimeChat

**RealtimeChat** — серверное приложение для чат-коммуникаций с поддержкой аутентификации пользователей, хранением сообщений,доставкой файлов, JWT-защитой, WebSocket для real-time общения, отображением онлайн-статуса, интеграцией с Redis, docker-окружением и swagger-документацией API.

---

## Возможности

- Регистрация и аутентификация пользователей (REST, JWT)
- Отправка сообщений (REST + WebSocket)
- Отправка файлов (вложений)
- WebSocket для получения новых сообщений в реальном времени
- Просмотр списка всех чатов с актуальным онлайн-статусом собеседников
- История сообщений c любым пользователем
- Передача статуса "онлайн/оффлайн" пользователей в реальном времени
- Работает с PostgreSQL (через Docker)
- Кеширование/статусы — Redis
- Документированное API через Swagger UI
- Контейнеризация (Docker, docker-compose)
- Простая запуск и настройка

---

## Быстрый старт

### 1. Клонируй репозиторий и перейди в папку

git clone https://github.com/yourname/RealtimeChat.git
cd RealtimeChat


### 2. Создай/проверь `.env` в корне

Пример:
SERVER_PORT=8080
DB_NAME=realtimechat
DB_USER=postgres
DB_PASSWORD=postgres
DB_PORT=5432
REDIS_HOST=redis
REDIS_PORT=6379
text

### 3. Запусти через docker-compose

docker-compose up --build
text

- Приложение на http://localhost:8080
- PostgreSQL и Redis стартуют автоматически
- Миграции применяются автоматически (см. папку `migrations`)

### 4. Открой Swagger UI для тестирования API

[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

---

## Разработка

### Запуск без Docker

1. Запусти PostgreSQL и Redis вручную.
2. Подготовь `.env` файл.
3. Запусти миграции из папки `migrations`.
4. Выполни:

go run cmd/realtime-chat/main.go

---

## Технологии

- Go (1.24+)
- PostgreSQL
- Redis
- Docker, docker-compose
- JWT (golang-jwt)
- Swagger (swaggo)
- WebSocket (gorilla/websocket)
