default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

# TESTARGS passes extra flags through to `go test`, e.g. a run filter for a
# scoped live pass: `make testacc TESTARGS='-run TestAccForwarder'`.
testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./... $(TESTARGS)

.PHONY: fmt lint test testacc build install generate
