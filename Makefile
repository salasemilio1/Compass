# Builds are reproducible (-trimpath, no VCS stamp, pinned toolchain in go.mod), so CI can rebuild
# the committed plugin binaries and fail if they differ from the source.
PLATFORMS := darwin-arm64 darwin-amd64 linux-arm64 linux-amd64
VERSION := $(shell sed -n 's/.*Version *= *"\(.*\)"/\1/p' internal/compass/init.go)
GOFLAGS_BUILD := -trimpath -buildvcs=false -ldflags='-s -w'

.PHONY: test build plugin verify-plugin dist smoke clean

test:
	go vet ./...
	go test ./...

build:
	CGO_ENABLED=0 go build $(GOFLAGS_BUILD) -o bin/compass ./cmd/compass

# The plugin ships these binaries; rebuild and commit them on every release.
plugin:
	@for p in $(PLATFORMS); do \
	  CGO_ENABLED=0 GOOS=$${p%-*} GOARCH=$${p#*-} go build $(GOFLAGS_BUILD) -o plugin/libexec/compass-$$p ./cmd/compass || exit 1; \
	done
	@ls -l plugin/libexec

verify-plugin:
	@rm -rf .verify && mkdir .verify
	@for p in $(PLATFORMS); do \
	  CGO_ENABLED=0 GOOS=$${p%-*} GOARCH=$${p#*-} go build $(GOFLAGS_BUILD) -o .verify/compass-$$p ./cmd/compass || exit 1; \
	  cmp -s .verify/compass-$$p plugin/libexec/compass-$$p || { echo "plugin/libexec/compass-$$p is stale: run make plugin and commit"; rm -rf .verify; exit 1; }; \
	done
	@rm -rf .verify && echo "plugin binaries match the source"

# Release archives for people and agents installing without the plugin or Go.
dist:
	@rm -rf dist && mkdir dist
	@for p in $(PLATFORMS); do \
	  mkdir -p dist/compass_$(VERSION)_$$p && \
	  CGO_ENABLED=0 GOOS=$${p%-*} GOARCH=$${p#*-} go build $(GOFLAGS_BUILD) -o dist/compass_$(VERSION)_$$p/compass ./cmd/compass && \
	  cp README.md dist/compass_$(VERSION)_$$p/ && \
	  tar -C dist -czf dist/compass_$(VERSION)_$$p.tar.gz compass_$(VERSION)_$$p && rm -rf dist/compass_$(VERSION)_$$p || exit 1; \
	done
	@cd dist && shasum -a 256 *.tar.gz > checksums.txt && cat checksums.txt

smoke: plugin
	sh tests/smoke.sh

clean:
	rm -rf bin/compass dist .verify
