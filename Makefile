.PHONY: all clean fmt test

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

fmt: *.go $(call rwildcard,cmd/,*.go) $(call rwildcard,fakes/,*.go)
	@find -not -path '*vendor*' -name \*.go -exec gofmt -s -w {} +

test:
	$(GO) test

clean:
	@rm -vf check in out
