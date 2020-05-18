SRC = $(shell find . -name '*.go' -type f)

.PHONY: all
all: test lint

.PHONY: test
test: ${SRC}
	go test ./...

.PHONY: lint
lint: ${SRC}
	go vet ./...
	golint -set_exit_status
