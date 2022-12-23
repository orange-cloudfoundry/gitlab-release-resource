.PHONY: all clean test

rwildcard = $(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

GO := go
CGO_ENABLED := 0
GOFLAGS := -ldflags=-s -w

all: check in out

check: *.go cmd/check/* $(call rwildcard,vendor/,*.go)
	GOFLAGS="${GOFLAGS}" CGO_ENABLED=${CGO_ENABLED} $(GO) build -o ./$@ ./cmd/$@

in: *.go cmd/in/* $(call rwildcard,vendor/,*.go)
	GOFLAGS="${GOFLAGS}" CGO_ENABLED=${CGO_ENABLED} $(GO) build -o ./$@ ./cmd/$@

out: *.go cmd/out/* $(call rwildcard,vendor/,*.go)
	GOFLAGS="${GOFLAGS}" CGO_ENABLED=${CGO_ENABLED} $(GO) build -o ./$@ ./cmd/$@

test:
	$(GO) test

clean:
	@rm -vf check in out
