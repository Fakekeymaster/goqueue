# binary name matches the CLI command name
BINARY = goqueue
GO = go

# to signify that all these are commands and not files
.PHONY = all buiild run clean test lint docker-up docker-down help

## all: Default when running make
all: build

# compile the binary for current OS/arch
#these flags strip debugging info and reduce binary size
#@ prints only output and no command
## build: build the binary
build:
	$(GO) build -ldflags="-s -w"  -o $(BINARY) .
	@echo "built ./$(BINARY)"

## run: run the binary
run: build
	./$(BINARY) server --workers 5 --port 8080

## test: run all tests with race detector enabled
test:
	$(GO) test ./... -v -race

## lint: run go vet to catch common mistakes
lint:
	$(GO) vet ./...

## clean: remove compiled binary
clean:
	rm -f $(BINARY)

## deps: download and tidy dependencies
deps:
	$(GO) mod tidy

## docker-up: build image and start Redis + goqueue together
docker-up:
	docker compose up --build

## docker-down: stop and remove all containers and volumes
docker-down:
	docker compose down -v

## submit: submit a sample job (server must already be running)
submit: build
	./$(BINARY) submit \
		--name "sample-job" \
		--type email_send \
		--priority high

## stats: show live queue metrics (server must already be running)
stats: build
	./$(BINARY) stats

## help: print this help message
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'