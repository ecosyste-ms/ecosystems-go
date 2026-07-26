.PHONY: generate test test-integration lint clean docs

SOURCEY_VERSION ?= 3.6.5

generate:
	go generate ./...

update-specs:
	vendir sync

test:
	go test -v ./...

test-integration:
	go test -v -tags=integration ./...

lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

clean:
	rm -f packages/packages.go repos/repos.go advisories/advisories.go commits/commits.go issues/issues.go
	rm -rf docs/dist docs/godoc.json

# Build the API reference site into docs/dist.
# The godoc snapshot is generated from the current source, not committed.
docs:
	cd docs && npm install --no-save --no-fund --no-audit sourcey@$(SOURCEY_VERSION)
	docs/node_modules/.bin/sourcey godoc -m . -o docs/godoc.json
	cd docs && node_modules/.bin/sourcey build
