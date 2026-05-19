.PHONY: \
	fmt \
	fmt-check \
	vet \
	lint \
	test \
	build \
	tidy \
	tidy-check \
	ci \
	release-check \
	release-snapshot \
	release

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test -race -cover ./...

build:
	go build -o bin/ ./...

tidy:
	go mod tidy

tidy-check: tidy
	git diff --exit-code -- go.mod go.sum

ci: fmt-check tidy-check vet lint test build

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean
