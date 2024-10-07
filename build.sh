bun install && \
  go install golang.org/x/tools/cmd/stringer@latest && \
  go install github.com/a-h/templ/cmd/templ@latest && \
  go generate ./... && \
  go run .