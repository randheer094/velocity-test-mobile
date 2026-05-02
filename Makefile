BINARY     := velocity-test-mobile
PKG        := ./...
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: build vet test lint tidy run clean list-tools install uninstall

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

# install matches the convention used by randheer094/velocity: move the
# built binary into $(INSTALL_DIR) (default ~/.local/bin). Override with:
#   make install INSTALL_DIR=/usr/local/bin
install: build
	mkdir -p $(INSTALL_DIR)
	mv $(BINARY) $(INSTALL_DIR)/$(BINARY)

uninstall:
	rm -f "$(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)
	rm -rf dist
