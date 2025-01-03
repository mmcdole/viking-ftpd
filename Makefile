.PHONY: build
build:
	go build -ldflags "-X main.version=$$(git describe --tags --always --dirty)" ./cmd/vkftpd
