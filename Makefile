.PHONY: generate test test-integration lint clean

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
