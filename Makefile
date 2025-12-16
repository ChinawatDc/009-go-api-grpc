.PHONY: proto tidy run-grpc run-rest

PROTO_DIR=proto
GEN_DIR=gen/go

proto:
	@echo ">> generating protobuf..."
	protoc -I $(PROTO_DIR) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/user/v1/user.proto
	@echo ">> done"

tidy:
	go mod tidy

run-grpc:
	go run ./cmd/grpc-server

run-rest:
	go run ./cmd/rest-api