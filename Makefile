NAME = status
BIN_DIR = bin

clean:
	rm -rf $(BIN_DIR)

deps:
	go mod download

build:
	mkdir -p $(BIN_DIR)
	go build -race -o $(BIN_DIR)/$(NAME)

run:
	go run main.go config.json

start:
	./$(BIN_DIR)/$(NAME) config.json

test:
	go test -race -v ./...

coverage:
	go test -race -cover -v ./...

vet:
	go vet ./...

lint:
	golint ./...

docker-build:
	docker build --rm -t willis7/status .

docker-run:
	docker run -i -t willis7/status
