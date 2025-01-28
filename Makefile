CODEGEN_DIR = internal/generated
SERVICE_PROTO = proto/service/service.proto

gen proto:
	mkdir -p internal/generated
	mkdir -p internal/generated/service
	protoc --proto_path=proto/service \
		--go_out=internal/generated/service --go_opt=paths=source_relative \
		--go-grpc_out=internal/generated/service --go-grpc_opt=paths=source_relative \
		service.proto

build:
	go build -o cmd/app/main cmd/main.go

update:
	./cmd/app/main gen -c config/config.yaml

run:
	air -c cmd/.air.toml