BINARY  := sweep
VERSION ?= 0.1.0
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test vet check clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY).exe .

test:
	go test ./...

vet:
	go vet ./...

check: vet test

clean:
	rm -rf dist
