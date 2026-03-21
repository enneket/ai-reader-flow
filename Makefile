.PHONY: all dev build build-win clean test

APP_NAME := ai-rss-reader
FRONTEND_DIR := frontend
BUILD_DIR := build

all: build

dev:
	wails dev

build:
	cd $(FRONTEND_DIR) && npm install
	cd $(FRONTEND_DIR) && npm run build
	wails build

build-win:
	cd $(FRONTEND_DIR) && npm install
	cd $(FRONTEND_DIR) && npm run build
	wails build -platform windows/amd64 -nsis

clean:
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -rf dist

test:
	go test ./...
