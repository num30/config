

test:
	go test -v ./...

# Go lint
lint:
	staticcheck .
