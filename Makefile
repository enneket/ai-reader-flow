.PHONY: all dev-go dev-frontend build build-docker up down logs test clean

APP_NAME := ai-rss-reader
FRONTEND_DIR := frontend
BUILD_DIR := build

all: build

dev-go:
	go run ./cmd/server

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

build: build-docker

build-docker:
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

test-api:
	curl http://localhost:8080/api/feeds

clean:
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -rf dist

test:
	go test ./...
