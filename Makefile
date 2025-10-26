.PHONY: all build run

all: build run

build:
	bun install && \
	go install golang.org/x/tools/cmd/stringer@latest && \
	go install github.com/a-h/templ/cmd/templ@v0.3.819 && \
	go generate ./... && \
	go run .

run:
	npx serve dist