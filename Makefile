NAME = status
PWD := $(MKPATH:%/Makefile=%)

clean:
	cd "$(PWD)"
	rm -rf vendor

install:
	glide install

build:
	go build -race -o $(NAME)

run:
	go run main.go config.json

start:
	./$(NAME) config.json

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
