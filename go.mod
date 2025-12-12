module github.com/Lina3386/telegram-bot

go 1.24.0

toolchain go1.24.7

require (
	github.com/Lina3386/auth v0.0.0-00010101000000-000000000000
	github.com/Lina3386/chat-server v0.0.0-00010101000000-000000000000
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/pressly/goose/v3 v3.21.0
	google.golang.org/grpc v1.75.1
)

replace (
	github.com/Lina3386/auth => ../auth
	github.com/Lina3386/chat-server => ../chat-server
)

require (
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
