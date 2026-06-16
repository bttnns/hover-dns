BINARY     := hover-dns
INSTALL_DIR := /usr/local/bin

.PHONY: build install uninstall clean

build:
	go build -o $(BINARY) .

install: build
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)

clean:
	rm -f $(BINARY)
