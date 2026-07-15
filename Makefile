.PHONY: generate test test-integration lint clean

OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1

generate:
	$(OAPI_CODEGEN) -generate types,client -package packages -o packages/packages.go specs/packages.yaml
	$(OAPI_CODEGEN) -generate types,client -package repos -o repos/repos.go specs/repos.yaml
	$(OAPI_CODEGEN) -generate types,client -package advisories -o advisories/advisories.go specs/advisories.yaml
	$(OAPI_CODEGEN) -generate types,client -package commits -o commits/commits.go specs/commits.yaml
	$(OAPI_CODEGEN) -generate types,client -package issues -o issues/issues.go specs/issues.yaml

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
