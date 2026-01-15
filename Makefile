.PHONY: generate test test-integration lint clean

OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

generate:
	$(OAPI_CODEGEN) -generate types,client -package packages specs/packages.yaml > packages/packages.go
	$(OAPI_CODEGEN) -generate types,client -package repos specs/repos.yaml > repos/repos.go

update-specs:
	curl -s "https://packages.ecosyste.ms/docs/api/v1/openapi.yaml" > specs/packages.yaml
	curl -s "https://repos.ecosyste.ms/docs/api/v1/openapi.yaml" > specs/repos.yaml

test:
	go test -v ./...

test-integration:
	go test -v -tags=integration ./...

lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

clean:
	rm -f packages/packages.go repos/repos.go
