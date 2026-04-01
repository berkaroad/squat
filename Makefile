go-test:
	go test -count=1 -coverprofile=coverage.out -timeout=30s ./...
	go tool cover -html=coverage.out -o coverage.html

go-bench:
	go test -run=none -count=1 -benchtime=10000x -benchmem -bench=. ./tests

go-staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck -f stylish -checks=all,-ST1000,-ST1023 ./...
