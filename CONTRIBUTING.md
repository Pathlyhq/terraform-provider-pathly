# Contributing

Thanks for helping with the Pathly Terraform provider
([pathlyhq.com](https://pathlyhq.com)).

## Prerequisites

- Go (see `go.mod`)
- Terraform ≥ 1.5 for `import` blocks / examples (CI uses a current 1.x)
- Optional: OpenTofu ≥ 1.6 (same provider binary)

## Build and unit tests

```sh
go build ./...
go vet ./...
go test ./...
```

Coverage on behaviour (excludes blocking `main.go`):

```sh
go test ./internal/... -coverprofile=covprofile
go tool cover -func=covprofile
```

Format before push:

```sh
gofmt -w $(git ls-files '*.go')
```

## Acceptance tests (`TF_ACC`)

Il n'existe **pas encore** de suite `TestAcc` qui lance un vrai
`terraform apply` / `destroy` contre une organisation Pathly. La CI exécute :

1. `go test ./internal/...` — cycle de vie CRUD contre une **API simulée** (`httptest`)
2. `terraform validate` sur les exemples (schéma, pas d'appel réseau)

La documentation historique ci-dessous décrit le *souhait* d'acceptance tests
réels : à activer seulement avec une org jetable et une clé à scopes minimaux.

```sh
# export TF_ACC=1
# export PATHLY_API_TOKEN="sp_…"   # org jetable uniquement
# go test ./internal/provider/... -run Acc -timeout 30m
```

Ne jamais pointer une org de production que vous ne pouvez pas muter.

## Secrets

- **Never** commit API tokens, GPG private keys, webhook secrets, or `.env` files.
- Prefer `PATHLY_API_TOKEN` / `PATHLY_API_URL` in the environment.
- Do not put `api_token` in example `.tf` files (it lands in plaintext state).
- CI signing keys stay in GitLab protected + masked variables only.

## Docs and examples

- English under `docs/` is canonical (Terraform Registry).
- Keep FR/ES trees in `translations/` aligned when adding or removing pages.
- Examples under `examples/` must `terraform validate` (see `.gitlab-ci.yml`).

## Local provider override

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "pathlyhq/pathly" = "/path/to/GOBIN"
  }
  direct {}
}
```

```sh
go install .
cd examples/minimal && terraform plan
```

## Author

Pathly · Simon Raynaud / keyral
