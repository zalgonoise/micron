
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run

test-unit: lint
	mkdir -p reports/coverage
	go test ./... -race -coverprofile=reports/coverage/coverage.out

test-integration:
	mkdir -p reports/coverage
	go test ./... -race -tags=integration -coverprofile=reports/coverage/coverage.out
