DB_URL="postgres://admin:admin@localhost:5433/crypto_chat?sslmode=disable"

NATS_SERVER=localhost:4222
STREAM_NAME=CHAT

PROTO_SRC = proto/chat.proto
PROTO_OUT = proto/chatpb

migrate-up:
	migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DB_URL)" down

migrate-new:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $${name}

get-msg:
	nats --server $(NATS_SERVER) stream get $(STREAM_NAME) $(MSG)

stream-info:
	nats stream info $(STREAM_NAME)

stream-ls:
	nats --server $(NATS_SERVER) stream ls

gen-proto:
	protoc \
    		--proto_path=./proto \
    		--go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
    		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
    		$(PROTO_SRC)