package ecosystems

//go:generate go run -modfile=./tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate types,client -package packages -o packages/packages.go specs/packages.yaml
//go:generate go run -modfile=./tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate types,client -package repos -o repos/repos.go specs/repos.yaml
//go:generate go run -modfile=./tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate types,client -package advisories -o advisories/advisories.go specs/advisories.yaml
//go:generate go run -modfile=./tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate types,client -package commits -o commits/commits.go specs/commits.yaml
//go:generate go run -modfile=./tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate types,client -package issues -o issues/issues.go specs/issues.yaml
