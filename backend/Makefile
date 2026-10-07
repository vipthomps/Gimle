run:
	cd cmd/gimle/ && \
	sudo \
	go run .

fmt:
	go fmt ./...

lint:
	golangci-lint run
	golint ./...

check: fmt lint

swag:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go --dir cmd/gimle,internal/api,internal/models,internal/portscan,internal/connectors,internal/docsource,internal/backup
