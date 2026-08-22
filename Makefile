.PHONY: run shoot build test vet fmt docker-build

run: ## corre el servidor en vivo (lee .env si existe)
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/server

shoot: ## corre cmd/shooter una vez, para todos los proyectos con web desplegada
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/shooter

build: ## compila ambos binarios a ./bin
	go build -o bin/server ./cmd/server
	go build -o bin/shooter ./cmd/shooter

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

docker-build:
	docker build -t repo-preview-service .
