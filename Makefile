.PHONY: build web test

web:
	cd web && npm ci && npm run build

build: web
	go build -o bin/codepath ./cmd/codepath

test:
	go test ./...
	cd web && npm test
