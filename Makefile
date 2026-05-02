BINARY := velocity-mcp-mobile
PKG    := ./...

.PHONY: build vet test lint tidy run clean list-tools

build:
	go build -trimpath -ldflags="-s -w" -o $(BINARY) .

vet:
	go vet $(PKG)

test:
	go test -count=1 $(PKG)

lint: vet
	gofmt -l . | tee /tmp/gofmt.out
	@! [ -s /tmp/gofmt.out ]

tidy:
	go mod tidy

run: build
	./$(BINARY)

list-tools: build
	./$(BINARY) --list-tools

clean:
	rm -f $(BINARY)
	rm -rf dist
