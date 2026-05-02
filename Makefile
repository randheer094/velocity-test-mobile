BINARY     := velocity-mcp-mobile
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

# install copies the built binary to $(INSTALL_DIR) (default ~/.local/bin).
# Override the destination by setting INSTALL_DIR, e.g.:
#   make install INSTALL_DIR=/usr/local/bin
install: build
	@mkdir -p "$(INSTALL_DIR)"
	install -m 0755 $(BINARY) "$(INSTALL_DIR)/$(BINARY)"
	@echo
	@echo "Installed: $(INSTALL_DIR)/$(BINARY)"
	@case ":$$PATH:" in \
	  *":$(INSTALL_DIR):"*) \
	    echo "OK: $(INSTALL_DIR) is already on your PATH." ;; \
	  *) \
	    echo "Note: $(INSTALL_DIR) is NOT on your PATH."; \
	    echo "      Add to your shell profile, e.g.:"; \
	    echo "        echo 'export PATH=\"$(INSTALL_DIR):\$$PATH\"' >> ~/.bashrc" ;; \
	esac

uninstall:
	rm -f "$(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)
	rm -rf dist
