.PHONY: generate test test-integration lint clean

OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1

generate:
	$(OAPI_CODEGEN) -generate types,client -package packages specs/packages.yaml > packages/packages.go
	$(OAPI_CODEGEN) -generate types,client -package repos specs/repos.yaml > repos/repos.go
	$(OAPI_CODEGEN) -generate types,client -package advisories specs/advisories.yaml > advisories/advisories.go
	$(OAPI_CODEGEN) -generate types,client -package commits specs/commits.yaml > commits/commits.go
	$(OAPI_CODEGEN) -generate types,client -package issues specs/issues.yaml > issues/issues.go

update-specs:
	curl -s "https://packages.ecosyste.ms/docs/api/v1/openapi.yaml" > specs/packages.yaml
	curl -s "https://repos.ecosyste.ms/docs/api/v1/openapi.yaml" > specs/repos.yaml
	curl -s "https://advisories.ecosyste.ms/docs/api/v1/openapi.yaml" > specs/advisories.yaml
	curl -s "https://commits.ecosyste.ms/docs/api/v1/openapi.yaml" > specs/commits.yaml
	curl -s "https://issues.ecosyste.ms/docs/api/v1/openapi.yaml" > specs/issues.yaml

test:
	go test -v ./...

test-integration:
	go test -v -tags=integration ./...

lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

clean:
	rm -f packages/packages.go repos/repos.go advisories/advisories.go commits/commits.go issues/issues.go
