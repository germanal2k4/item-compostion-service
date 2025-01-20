# Item Composition Service

## Description

## Usage

## File architecture

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
│   └── services/                   # Business logic
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