.PHONY: all
all: test build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -count=1 ./...

.PHONY: build
build:
	go build -o static-pv-releaser -race -v .

.PHONY: clean
clean:
	go clean ./...
