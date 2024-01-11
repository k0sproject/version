GO_SRCS := $(shell find . -type f -name '*.go' -a ! \( -name 'zz_generated*' -o -name '*_test.go' \))
TAG_NAME = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null || echo "dev")
PREFIX = /usr/local
LD_FLAGS = -s -w -X github.com/k0sproject/version/internal/version.Version=$(TAG_NAME)
BUILD_FLAGS = -trimpath -a -tags "netgo,osusergo,static_build" -installsuffix netgo -ldflags "$(LD_FLAGS) -extldflags '-static'"

k0s_sort: $(GO_SRCS)
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ ./cmd/k0s_sort

PLATFORMS := linux-amd64 linux-arm64 linux-arm darwin-amd64 darwin-arm64 windows-amd64
BIN_PREFIX := k0s_sort-
bins := $(foreach platform, $(PLATFORMS), bin/$(BIN_PREFIX)$(platform))
$(bins):
	$(eval temp := $(subst -, ,$(subst $(BIN_PREFIX),,$(notdir $@))))
	$(eval OS := $(word 1, $(subst -, ,$(temp))))
	$(eval ARCH := $(word 2, $(subst -, ,$(temp))))
	$(eval EXT := $(if $(filter $(OS),windows),.exe,))
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@$(EXT) ./cmd/k0s_sort

bin/sha256sums.txt: $(bins)
	sha256sum -b $(bins) | sed 's|bin/||' > $@

build-all: $(bins) bin/sha256sums.txt

.PHONY: install
install: k0s_sort
	install -d $(DESTDIR)$(PREFIX)/bin/
	install -m 755 k0s_sort $(DESTDIR)$(PREFIX)/bin/

.PHONY: test
test:
	go clean -testcache && go test -count=1 -v ./...

.PHONY: clean
clean:
	rm -rf k0s_sort bin

