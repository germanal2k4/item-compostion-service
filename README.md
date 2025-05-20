# Item Composition Service

Сервис для композиции элементов с поддержкой шаблонов и провайдеров.

## Требования

- Go 1.21 или выше
- Docker и Docker Compose
- Make
- Protocol Buffers (protoc)

## Установка

1. Клонируйте репозиторий:
```bash
git clone <repository-url>
cd item-composition-service
```

2. Установите зависимости:
```bash
go mod download
```

## Запуск сервиса

### Локальный запуск

1. Сгенерируйте proto-файлы:
```bash
make gen proto
```

2. Соберите проект:
```bash
make build
```

3. Запустите сервис:
```bash
make run
```

### Запуск в Docker

1. Соберите и запустите все сервисы:
```bash
make deploy
```

Это запустит:
- Основной сервис на порту 3030 (gRPC) и 8080 (HTTP)
- Jaeger для трейсинга (порт 16686)
- Prometheus для метрик (порт 9090)
- Grafana для визуализации (порт 3000)
- Elasticsearch (порт 9200)

## Доступ к сервисам

- Основной сервис: `localhost:3030` (gRPC), `localhost:8080` (HTTP)
- Jaeger UI: `http://localhost:16686`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (логин: admin, пароль: admin)
- Elasticsearch: `http://localhost:9200`

## Конфигурация

Конфигурация сервиса находится в директории `config/`:
- `config.yaml` - основная конфигурация
- `config_local.yaml` - локальная конфигурация для разработки

## Обновление шаблонов

Для обновления шаблонов используйте команду:
```bash
make update
```

## Разработка

### Структура проекта

- `cmd/` - точка входа в приложение
- `internal/` - внутренний код приложения
- `pkg/` - публичные пакеты
- `proto/` - proto-файлы
- `config/` - конфигурационные файлы
- `deployment/` - файлы для развертывания

### Полезные команды

- `make gen proto` - генерация proto-файлов
- `make build` - сборка проекта
- `make run` - запуск сервиса
- `make docker_build` - сборка Docker-образа
- `make deploy` - полное развертывание всех сервисов

## Мониторинг

Сервис интегрирован с:
- Jaeger для трейсинга
- Prometheus для сбора метрик
- Grafana для визуализации
- Elasticsearch для хранения логов

## Логирование

Логи доступны через:
- Docker logs: `docker logs <container-name>`
- Elasticsearch: `http://localhost:9200`

```
item-composition-service/
├── api/                            # Openapi specifications of HTTP clients
|
├── cmd/                         
│   └── main.go                     # Entrypoinyt of program
|
├── config/                         
|   ├── config.yaml                 # Program configuration files
│   └── envs                        # Enviroments 
|
├── docs/                           # Programm documentation
|
├── internal/
│   ├── config/                     # Application configuration
│   ├── entities/                   # Domain types
│   ├── generated/                  # Codegen files
│   ├── repository/                 # Database access layer
│   ├── server/                     # gRPC server implementation
│   ├── services/                   # Business logic
│   └── setup/                      # Application setup
|
├── migrations/                     # Database migrations
|
├── pkg/                            # Application libraries
|        
├── proto   
│   ├── item-composition-service    # Protobuf file of Item Composition Service
│   └── clients                     # Protobuf files of gRPC clients  
|   
├── test                            # Tests
├── docker-compose.yaml             
├── Dockerfile                      # Deployment file
├── go.mod                          # Go module dependencies
├── Makefile                        # Launch scripts
└── README.md                       # Project overview
```