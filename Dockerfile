# ===== STAGE 1: BUILD =====
FROM golang:1.24-alpine AS builder

WORKDIR /app

# ✅ Установим необходимые зависимости
RUN apk add --no-cache git build-base

# ✅ Копируем go.mod и go.sum
COPY go.mod go.sum ./

# ✅ Загружаем зависимости
RUN go mod download

# ✅ Копируем исходный код
COPY . .
# Скопируй .env внутрь образа
COPY .env .env


# ✅ Собираем бинарь
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot cmd/main.go

# ===== STAGE 2: RUNTIME =====
FROM alpine:latest

WORKDIR /root/

# ✅ Устанавливаем ca-certificates для TLS
RUN apk --no-cache add ca-certificates

# ✅ Копируем бинарь из builder
COPY --from=builder /app/bot .

# ✅ Открываем порт
EXPOSE 8081

# ✅ Запускаем бота
CMD ["./bot"]
