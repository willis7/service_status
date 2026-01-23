NAME = status
PWD := $(MKPATH:%/Makefile=%)
BIN_DIR = bin

clean:
	cd "$(PWD)"
	rm -rf vendor
	rm -rf $(BIN_DIR)

install:
	glide install

build:
	mkdir -p $(BIN_DIR)
	go build -race -o $(BIN_DIR)/$(NAME)

run:
	go run main.go config.json

start:
	./$(BIN_DIR)/$(NAME) config.json

test:
	go test -race -v $(shell glide novendor)

coverage:
	go test -race -cover -v $(shell glide novendor)

vet:
	go vet $(shell glide novendor)

lint:
	golint $(shell glide novendor)

docker-build:
	docker build --rm -t willis7/status .

docker-run:
	docker run -i -t willis7/status
