FROM golang:1.23.5-alpine AS base

WORKDIR /app

EXPOSE 3030

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/.air.toml                          ./cmd/.air.toml
COPY --chmod=0755 cmd/app/main              ./cmd/app/main
COPY api                                    ./api
COPY proto                                  ./proto
COPY config                                 ./config

CMD ["air", "-c", "cmd/.air.toml"]
