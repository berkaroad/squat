go-test:
	go test -count=1 -coverprofile=coverage.out -timeout=30s ./...
	go tool cover -html=coverage.out -o coverage.html
	