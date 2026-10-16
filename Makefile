run:
	go run ./cmd/kitchen

test:
	go test ./...

build:
	go build ./cmd/kitchen
	go build ./cmd/restaurant

docker-up:
	docker compose up --build

docker-down:
	docker compose down
