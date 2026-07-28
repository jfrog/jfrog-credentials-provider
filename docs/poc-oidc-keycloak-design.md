# PoC: Generic / Keycloak OIDC → JFrog Credential Provider Flow

## What this proves

`jfrog-credentials-provider` only had cloud-specific login handlers (AWS,
Azure, Google) before this PoC. This adds a small **generic OIDC** handler
and proves, with `go test` unit tests that pass in GitHub Actions, that the
full flow works with **no managed Kubernetes cluster and no cloud account**:

```
CredentialProviderRequest (id_token) --> JFrog OIDC token exchange --> CredentialProviderResponse
```

The JFrog-side token exchange (`handlers.ExchangeOidcArtifactoryToken`,
`/access/api/v1/oidc/token`) was already cloud-agnostic — it hardcodes
`provider_type: "Generic OpenID Connect"` for every caller. The actual gap
was on the "get an id_token" side, which this PoC closes with
`internal/handlers/generic.go` + `cloud_provider=generic` wiring in
`internal/provider/provider.go`.

## The two test tiers, and exactly what each one covers

### Tier A — pure unit test (`internal/handlers/jfrog_exchange_test.go`)

- **Zero network.** Artifactory is stood in for by an in-process
  `httptest.NewTLSServer`. The id_token is a hand-typed opaque string
  (`"fake.header.payload"`) — the provider code never parses or verifies it,
  it only forwards it as `subject_token`, so a real JWT isn't required to
  prove the code path.
- Asserts the whole D1 chain: `CredentialProviderRequest` + id_token →
  `GetGenericIdentityToken` → `ExchangeOidcArtifactoryToken` (request shape:
  grant type, subject token type, provider type, provider name, audience,
  subject token all asserted) → `BuildCredentialProviderResponse` → the
  resulting JSON has the right registry key and the stub's exchanged token
  as the password.
- `TestGenericAuth_EndToEnd` additionally drives `provider.GenericAuth`, the
  function the kubelet path itself calls for `cloud_provider=generic`, so the
  orchestration under test is production's rather than the test body's. It
  reads `jfrog_oidc_provider_name` / `jfrog_oidc_audience` from the
  environment exactly as the deployed binary does, which means a swapped
  argument or a renamed env var fails the build.
- Also covers negative paths: stub returns 401 → error surfaces; empty
  `ServiceAccountToken` → `GetGenericIdentityToken` errors before any HTTP
  call is made; either OIDC env var missing → `GenericAuth` errors and
  Artifactory is never contacted.
- `internal/utils/validate_generic_test.go` covers the `CloudProviderGeneric`
  branch of `ValidateJfrogProviderConfig`, the other half of the D2 wiring.
- Runs in the `unit-tests` job of `.github/workflows/poc-unit-tests.yml` on
  a GitHub-hosted `ubuntu-latest` runner. No Docker, no secrets, no cloud
  credentials.

### Tier B — Keycloak-backed (`internal/handlers/generic_keycloak_test.go`, build tag `keycloak_integration`)

- Boots a real Keycloak container from **config-as-code**
  (`test/keycloak/docker-compose.yml` + `test/keycloak/realm-export.json`,
  imported via `start-dev --import-realm` — not clicked together in the
  admin UI).
- Performs a real Resource Owner Password Credentials grant against the
  imported `jfrog-poc` realm and gets back a real, Keycloak-signed
  `id_token` JWT (sanity-checked here for `iss`/`aud` claims, without
  needing a JWT library — the payload segment is just base64-decoded JSON).
- `poc-user` carries `firstName` and `lastName` for a non-obvious reason:
  since Keycloak 24 the declarative User Profile is always on and marks both
  as required, so a user lacking them gets the `VERIFY_PROFILE` required
  action injected at login and the password grant is rejected with
  `invalid_grant` / "Account is not fully set up". Setting the user's
  `requiredActions` to `[]` is not sufficient on its own, because the action
  is recomputed from profile completeness on every authentication.
- Feeds that real id_token through the **exact same** `provider.GenericAuth`
  → `BuildCredentialProviderResponse` path as Tier A, against the same kind of
  stub Artifactory server — Artifactory itself is still not required to be
  real. The exchange request is asserted just as strictly as in Tier A
  (grant type, subject token type, provider type, provider name, audience,
  and that `subject_token` is byte-for-byte the JWT Keycloak minted), so this
  tier proves the real token arrives intact rather than only that some token
  did.
- Readiness is decided by polling the realm's OIDC discovery document, not a
  container healthcheck: it confirms the realm was actually *imported*, not
  merely that Keycloak's process started. The bound is 120s and can be raised
  with `KEYCLOAK_READY_TIMEOUT_SECONDS` on slow runners.
- Gated behind the `keycloak_integration` build tag so a plain
  `go test ./...` never depends on Docker being present; runs as its own
  `keycloak-integration-test` job in CI, on `ubuntu-latest` with Docker
  available out of the box — still no managed cluster, no cloud account.

## What this PoC does **not** cover (said plainly, not glossed over)

- **A real Artifactory server.** Both tiers stub the JFrog OIDC token
  endpoint. The exchange code itself (`ExchangeOidcArtifactoryToken`) is
  unmodified and identical to what AWS/Azure/Google already use in
  production, but this PoC does not prove connectivity to a live
  Artifactory instance.
- **The real kubelet-invoked binary protocol end-to-end.** The compiled
  binary reads `CredentialProviderRequest` from stdin and writes
  `CredentialProviderResponse` to stdout via `provider.StartProvider`; the
  tests exercise the same functions directly (`GetGenericIdentityToken`,
  `ExchangeOidcArtifactoryToken`, `BuildCredentialProviderResponse`) rather
  than piping through the actual compiled binary's stdin/stdout, because
  `provider.StartProvider`'s error paths call `logs.Exit`
  (`os.Exit`), which would kill the test process, and its logger writes to
  `/var/log/jfrog-credentials-provider/...`, which requires root. This
  mirrors the existing untested state of `handleAzureAuth`/`handleGoogleAuth`
  — not a gap introduced by this PoC.
- **Helm chart wiring for `cloud_provider=generic`.** `helm/values.yaml`,
  `helm/templates/_helpers.tpl`, and `helm/templates/validations.yaml` still
  only know about `aws`/`azure`/`gcp`. Making `generic` deployable via the
  Helm chart is follow-up work, not part of this PoC.
- **`add-provider-config` / `CreateProviderConfigFromEnv` env-var plumbing**
  for generic mode (the standalone provider-config-file generation path).
  Only `ValidateJfrogProviderConfig`'s validation switch was extended.
- **Any real or managed Kubernetes cluster.** No EKS/AKS/GKE, no OpenShift,
  no local k3s/minikube. (A local k3s/minikube smoke test remains a
  stretch add-on if time allows at the very end of the 15 days — it is not
  a substitute for anything above.)
- **Keycloak realm secrets in this repo are test-only** (`poc-test-secret-not-for-prod`,
  a hardcoded test-user password) and must never be reused outside this PoC.
- **Audience is not split into two settings.** `handleGenericAuth` uses one
  value, `jfrog_oidc_audience`, for both the audience the id_token was minted
  for and the `audience` field of the JFrog token exchange. `handleGoogleAuth`
  does the same, but `handleAzureAuth` deliberately keeps them apart
  (`azure_app_audience` for the IdP, `jfrog_token_audience` defaulting to
  `*@*` for the exchange). They are different things, and a deployment whose
  Artifactory identity mapping expects a token audience unlike the IdP
  audience will need them separated. That is not exercised here because both
  tiers stub Artifactory, so nothing validates the `audience` field.

## Files

| Purpose | Path |
|---|---|
| Generic OIDC id_token extraction | `internal/handlers/generic.go` |
| Cloud-provider wiring | `internal/provider/provider.go` (`GenericAuth`, `handleGenericAuth`, `BuildCredentialProviderResponse`), `internal/utils/utils.go` (`CloudProviderGeneric`, validation) |
| Tier A test | `internal/handlers/jfrog_exchange_test.go` |
| Tier B test | `internal/handlers/generic_keycloak_test.go` |
| Config validation test | `internal/utils/validate_generic_test.go` |
| Keycloak config-as-code | `test/keycloak/realm-export.json`, `test/keycloak/docker-compose.yml` |
| CI | `.github/workflows/poc-unit-tests.yml` |

## Running it yourself

```sh
# Tier A - no Docker needed
go test ./...

# Tier B - requires Docker
docker compose -f test/keycloak/docker-compose.yml up -d
go test -tags=keycloak_integration ./internal/handlers/... -run TestGenericOidcFlow_KeycloakBacked -v
docker compose -f test/keycloak/docker-compose.yml down -v
```
