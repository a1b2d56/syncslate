GO := "C:\Program Files (x86)\Go\bin\go.exe"
BINARY_NAME := bin/syncslate.exe

.PHONY: all build run test clean docker-build

all: build

build:
	@if not exist bin mkdir bin
	$(GO) build -o $(BINARY_NAME) ./cmd/syncslate

run: build
	./$(BINARY_NAME)

test:
	$(GO) test ./... -v -count=1

clean:
	@if exist bin rmdir /s /q bin
	@if exist syncslate.db del /f /q syncslate.db

docker-build:
	docker build -t syncslate-backend:latest .
