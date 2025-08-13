
.PHONY: start-infra
start-infra:
	docker compose up -d

.PHONY: produce-message
produce-message:
	go run main.go

.PHONY: generate-proto
generate-proto:
	protoc --go_out=. --go-grpc_out=. --proto_path=. message_reversed.proto

.PHONY: start-connect-consumer
start-connect-consumer:
	rpk connect -c consumer.yaml