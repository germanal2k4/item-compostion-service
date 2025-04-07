FROM ubuntu AS builder

WORKDIR /app

RUN apt-get update && \
    apt-get install -y golang-go && \
    apt-get install -y protobuf-compiler && \
    apt-get install -y protoc-gen-go && \
    apt-get install -y protoc-gen-go-grpc && \
    apt-get install -y ca-certificates

COPY go.mod /app/go.mod
COPY go.sum /app/go.sum

COPY cmd       /app/cmd
COPY internal  /app/internal
COPY pkg       /app/pkg
COPY proto/service /app/proto

RUN mkdir -p /app/internal/generated && \
    mkdir -p /app/internal/generated/service && \
    protoc --proto_path=/app/proto \
    --go_out=/app/internal/generated/service --go_opt=paths=source_relative \
    --go-grpc_out=/app/internal/generated/service --go-grpc_opt=paths=source_relative \
    service.proto

RUN go build -o /opt/bin/item-composition-service /app/cmd/main.go

FROM ubuntu

EXPOSE 3030
EXPOSE 8080

RUN apt-get update && apt-get install -y supervisor && apt-get install -y logrotate

COPY api                                        /app/api
COPY proto                                      /app/proto
COPY config/config.yaml                         /etc/item-composition-service/config.yaml

COPY --chmod=0644 deployment/logrotate.d/logrotate         /etc/logrotate.d/item-composition-service
COPY deployment/supervisor/supervisord.conf                /etc/supervisor/supervisord.conf
COPY --chmod=0755 deployment/pre_start.sh                  /opt/bin/pre_start

COPY --from=builder /opt/bin/item-composition-service /opt/bin/item-composition-service

RUN /opt/bin/pre_start

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/supervisord.conf"]
