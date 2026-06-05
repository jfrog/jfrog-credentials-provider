# YAML-Driven E2E Test Framework - Context for Continuation

## What Was Done (This Session)

### 1. Removed redundant wrapper scripts
Deleted `aws.sh`, `azure.sh`, `gcp.sh` — they were thin wrappers around `runner.sh` that the CI workflow (`helm.test.yml`) already calls directly. No references remain.

### 2. Added `--provider` flag to `runner.sh`
For convenient local execution without remembering glob paths:
```bash
source build/test/env && bash build/test/runner.sh --provider aws
bash build/test/runner.sh --provider azure
bash build/test/runner.sh --provider all --parallel
```

### 3. Updated `env` file with all CI-injected vars
Added commented-out blocks (`>>> Uncomment for local runs <<<`) for every env var that CI injects via secrets/inputs:
- `DOWNLOAD_URL` (shared)
- `AWS_NODE_ROLE_ARN`, `AWS_SUBNET_IDS` (AWS)
- `AZURE_APP_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_NODEPOOL_CLIENT_ID`, `JFROG_OIDC_PROVIDER_NAME` (Azure)
- `GCP_SERVICE_ACCOUNT_EMAIL`, `JFROG_OIDC_PROVIDER_NAME` (GCP)

### 4. Auto-inject nodeSelector into Helm values
`runner.sh` now appends `.nodeSelector."jfrog-test" = "<case-name>"` to every step's overrides, so the chart's DaemonSet pods land on the correct test node group.

### 5. Added `chartSource: released` support for upgrade testing
- `resolve_chart_ref()` in `runner.sh` reads `.steps[N].helm.chartSource` from the YAML
- `released` → does `helm repo add jfrog https://charts.jfrog.io/` (once), returns `jfrog/jfrog-credential-provider`
- `local` or omitted → returns `${REPO_ROOT}/helm`
- Optional `chartVersion` field supported for pinning
- `helm_install()` and `helm_upgrade()` in `helper.sh` now accept an optional 4th arg `CHART_REF` (defaults to local)
- Stdout from `helm repo add/update` redirected to stderr so it doesn't pollute the `$()` capture

### 6. Fixed `envsubst` incompatibility
`envsubst` doesn't support `${VAR:-default}` syntax — it passes it through as a literal string. Replaced all `${VAR:-default}` patterns in YAML case files with hardcoded values (the defaults were already static infrastructure values like VM sizes).

### 7. Disabled AWS CLI pager
Added `export AWS_PAGER=""` in `helper.sh` globals to prevent the AWS CLI from opening `less`/vi, which blocks CI and local terminal output.

### 8. Fixed AMI type casing
`AL2023_X86_64_STANDARD` → `AL2023_x86_64_STANDARD` (AWS API requires lowercase `x86_64`).

### 9. Azure VMSS identity assign fallback
`az vmss identity assign` has a known bug that produces duplicate JSON keys when the VMSS already has identities. Added a fallback to `az vmss update --set`.

### 10. Replaced Azure VMSS identity assignment with `az rest`
Removed both `az vmss identity assign` (duplicate JSON bug) and the broken `az vmss update --set` fallback (dot-splitting on `Microsoft.ManagedIdentity` in resource IDs). Replaced with a single `az rest --method PATCH` call that sends the identity as a JSON body, bypassing all CLI parsing issues.

## Architecture

```
build/test/
  runner.sh              # YAML-driven test executor with --provider flag
  helper.sh              # Shared bash primitives (cloud CLI wrappers, helm, pods)
  env                    # Config: exported values + commented-out local-only vars
  cases/                 # One YAML per test case
    aws-assume-role.yaml
    aws-assume-role-upgrade.yaml   # install released → upgrade to local
    aws-projected-sa.yaml
    azure-oidc.yaml                # amd64 (Standard_D2ds_v4)
    azure-projected-token.yaml     # arm64 (Standard_D2pds_v5)
    gcp-oidc.yaml
    gcp-oidc-arm64.yaml
```

## YAML Test Case Schema

```yaml
name: aws-assume-role          # REQUIRED - derives namespace, release name, node labels
provider: aws                  # REQUIRED - aws | azure | gcp

nodeGroup:                     # Provider-specific node group config
  # AWS: instanceType, amiType
  # Azure: vmSize, nodeCount, identityClientId
  # GCP: machineType, numNodes, serviceAccountEmail

steps:
  - action: install            # install | upgrade
    helm:
      chartSource: released    # "released" = from jfrog helm repo; omit = local ./helm
      chartVersion: "0.1.0"    # optional, pin released chart version
      baseValues: examples/aws-values.yaml
      overrides:               # yq expressions, env vars expanded via envsubst
        - '.providerConfig[0].artifactoryUrl = "${ARTIFACTORY_URL}"'
    verify:
      image: "${TEST_IMAGE}"
      projectedToken: false    # true creates a projected SA + annotates it
```

## Auto-generated from `name`

Given `name: aws-assume-role` and `RUN_ID=1741234567`:

| Field          | Value                           | Rule                                         |
|----------------|---------------------------------|----------------------------------------------|
| namespace      | `aws-assume-role-41234567`      | `${name}-${SHORT_RUN_ID}` (last 8 of RUN_ID) |
| release_name   | `aws-assume-role`               | `${name}`                                    |
| ng_name        | `aws-assume-role-41234567`      | `${name}-${SHORT_RUN_ID}`                    |
| ng_name (azure)| `azur41234567`                  | first 4 alphanum of name + SHORT_RUN_ID      |
| nodeSelector   | `jfrog-test: aws-assume-role`   | auto-injected into helm values               |

## Branch Info

- Repo: `jfrog-credentials-provider`
- Branch: `feature/INST-19278`
- Changes are uncommitted.
