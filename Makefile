.PHONY: all build run run-relaydns

all: build run

build:
	bun install && \
	go install golang.org/x/tools/cmd/stringer@latest && \
	go install github.com/a-h/templ/cmd/templ@v0.3.819 && \
	go generate ./... && \
	go run .

run:
	npx serve dist

run-relaydns:
	go run ./cmd/relaydns-proxy --dir dist --port 8081 --name gosuda-blog --server-url http://relaydns.gosuda.org
