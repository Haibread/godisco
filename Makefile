.PHONY: all build test test-race vet lint fmt cover clean docker tidy help

BINARY := godisco
PKG    := ./...
CMD    := ./app/godisco
COVER  := coverage.out

all: vet test build

build:
	go build -v -o $(BINARY) $(CMD)

test:
	go test -v $(PKG)

test-race:
	go test -race -v $(PKG)

vet:
	go vet $(PKG)

lint:
	golangci-lint run $(PKG)

fmt:
	gofmt -s -w .

cover:
	go test -race -coverprofile=$(COVER) -covermode=atomic $(PKG)
	go tool cover -func=$(COVER)

tidy:
	go mod tidy

docker:
	docker build -t godisco:dev .

clean:
	rm -f $(BINARY) $(COVER)

help:
	@echo "Targets:"
	@echo "  build      - compile binary"
	@echo "  test       - run tests"
	@echo "  test-race  - run tests with race detector"
	@echo "  vet        - run go vet"
	@echo "  lint       - run golangci-lint (must be installed)"
	@echo "  fmt        - format code with gofmt"
	@echo "  cover      - run tests with coverage report"
	@echo "  tidy       - run go mod tidy"
	@echo "  docker     - build Docker image"
	@echo "  clean      - remove build artifacts"
