BINARY  := mping
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

TARGETS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	linux/arm \
	windows/amd64 \
	freebsd/amd64

.PHONY: build dist clean help

build:
	go build $(LDFLAGS) -o $(BINARY) .

dist: $(TARGETS)

$(TARGETS):
	$(eval OS   := $(word 1, $(subst /, ,$@)))
	$(eval ARCH := $(word 2, $(subst /, ,$@)))
	$(eval EXT  := $(if $(filter windows,$(OS)),.exe,))
	$(eval OUT  := dist/$(BINARY)-$(OS)-$(ARCH)$(EXT))
	GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) -o $(OUT) .
	@echo "  $(OUT)"

clean:
	rm -f $(BINARY) $(BINARY).exe
	rm -rf dist/

help:
	@echo "Targets:"
	@echo "  build   build for current platform"
	@echo "  dist    build for all platforms into dist/"
	@echo "  clean   remove build artifacts"
