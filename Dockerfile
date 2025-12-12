FROM golang:1.24-alpine AS builder-base

WORKDIR /workspace
COPY ./auth ./auth
COPY ./chat-server ./chat-server
COPY ./telegram-bot ./telegram-bot

WORKDIR /workspace/telegram-bot
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder-base /workspace/telegram-bot/bot .

EXPOSE 8081
CMD ["./bot"]
