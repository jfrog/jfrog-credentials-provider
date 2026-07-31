# Cloud provider unit tests

This directory contains black-box tests for the AWS, Azure, and Google
credential-provider flows.
The tests import the production packages but remain separate from the existing
tests under `internal/`.

## Test boundaries

- AWS EC2 IMDS, Azure IMDS/Azure AD, Google metadata/IAM Credentials, and JFrog
  Access calls are routed to in-process `httptest` servers.
- AWS SDK calls use small injected client interfaces and deterministic fakes.
- SigV4a signing uses the production signer.
- No cloud account, Kubernetes cluster, Docker daemon, or external network is
  required.

## Covered flows

- AWS: `assume_role`, `web_identity`, `assume_external_role`, and
  `cognito_oidc`.
- Azure: managed-identity assertion, Azure AD token exchange, and JFrog OIDC
  exchange.
- Google: metadata service-account token, IAM Credentials ID token generation,
  and JFrog OIDC exchange.
- Generic OIDC: covered by the existing tests under `internal/handlers/`; those
  files are intentionally left unchanged.

## Why Keycloak is not used

Keycloak implements OAuth 2.0 and OpenID Connect. It is useful for the generic
OIDC provider and for testing consumers that validate JWT signatures through an
OIDC discovery document and JWKS.

The cloud handlers call provider-specific APIs: AWS IMDS/STS/Cognito, Azure
IMDS/Azure AD, and Google metadata/IAM Credentials. Keycloak cannot emulate
those protocols. The tests therefore mock the actual HTTP and SDK boundaries
used by each production handler.

The existing Keycloak-backed generic OIDC test remains a separate Docker-based
integration test.

## Running

```sh
go test ./unit-tests/... -v
```

The repository's existing `go test ./...` CI command discovers this package
automatically.
