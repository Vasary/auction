export DATABASE_URL=postgres://auction:NKI88uprktHXKXVb@10.10.0.4:5432/auction_db?sslmode=disable
export AMQP_URL=amqp://rabbit:password@10.10.0.4:5672/

.PHONY: build-ui run-server run docker-build test

test:
	go test ./...

build-ui:
	cd ui && npm install && npm run build

run-server:
	go run ./cmd/server

run: build-ui run-server

docker-build:
	docker build -t auction-core .
