lint:
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

test:
	mkdir -p reports/coverage
	go test ./... -coverprofile=reports/coverage/coverage.out

test-integration:
	mkdir -p reports/coverage
	go test ./... -race -tags=integration -coverprofile=reports/coverage/coverage.out
