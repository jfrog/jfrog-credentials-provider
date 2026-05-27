# OpenShift testing plans

Manual QA scripts for the JFrog Kubelet Credential Provider on **OpenShift 4.21+** with **projected service account** identity (`tokenAttributes`).

| Cloud | Script | Example values | Scratch values (optional) |
|-------|--------|----------------|---------------------------|
| AWS | [openshift-test-plan-aws.sh](./openshift-test-plan-aws.sh) | [openshift-aws-projected-sa-values.yaml](../examples/openshift-aws-projected-sa-values.yaml) | [test-aws.yaml](./test-aws.yaml) |
| Azure | [openshift-test-plan-azure.sh](./openshift-test-plan-azure.sh) | [openshift-azure-projected-sa-values.yaml](../examples/openshift-azure-projected-sa-values.yaml) | [test-azure.yaml](./test-azure.yaml) |

Full setup reference: [OpenShift.md](../OpenShift.md)

## Prerequisites

- OpenShift **4.21+** (`tokenAttributes` required)
- `oc` logged in as cluster-admin
- `helm` 3.x
- Clone of this repository
- Artifactory **admin** access token
- `TEST_IMAGE` — any image in a repo the mapped Artifactory user can pull (scripts use `sleep infinity`, not the image entrypoint)

**Azure:** `az` CLI, `RESOURCE_GROUP` for the ARO cluster  
**AWS:** `aws` CLI  

Install the chart from the **local `./helm` chart** (not the published JFrog chart repo).

## Usage

```bash
chmod +x scratch/openshift-test-plan-*.sh

# Required for all clouds (hostname only — no https://)
export ARTIFACTORY_URL="your-instance.jfrog.io"
export ARTIFACTORY_ADMIN_TOKEN="..."
export TEST_IMAGE="${ARTIFACTORY_URL}/docker-local/your-image:tag"

# Full test (no teardown)
bash scratch/openshift-test-plan-azure.sh

# Full test, then remove created resources
bash scratch/openshift-test-plan-azure.sh all --cleanup

# Teardown only
bash scratch/openshift-test-plan-azure.sh --cleanup

# Single phase
bash scratch/openshift-test-plan-azure.sh --phase 3

# Use scratch values instead of examples/
export VALUES_FILE=scratch/test-azure.yaml
export OIDC_PROVIDER_NAME=aro-workload-identity   # must match values + Artifactory
bash scratch/openshift-test-plan-azure.sh
```

## Phase overview

| Phase | AWS | Azure | GCP |
|-------|-----|-------|-----|
| 0 | Prerequisites (OIDC issuer, IAM OIDC provider check) | Prerequisites (pod-identity-webhook) | Prerequisites (issuer, WIF provider path) |
| 1 | IAM role + workload SA | Managed identity + federated credential + SA | Google service account + WIF binding + SA annotations |
| 2 | Artifactory IAM role mapping | Artifactory OIDC provider + mapping | Artifactory OIDC provider + mapping |
| 3 | Helm install (namespace pre-created; no `--create-namespace`) | Helm install | Helm install |
| 4 | Verify namespace / DaemonSet / SCC | Verify namespace / pods | Verify namespace / pods |
| 5 | Verify node plugin + merged config | Verify node plugin + merged config | Verify node plugin + merged config |
| 6 | Injector logs | Injector logs | Injector logs |
| 7 | Positive image pull | Positive image pull | Positive image pull |
| 8 | Negative pull (no WI annotations) | Negative pull | Negative pull |

## What each script creates

| Cloud | Created by script | `--cleanup` removes |
|-------|-------------------|---------------------|
| AWS | IAM role, `jfrog-pull-test` namespace, Helm release in `jfrog`, Artifactory IAM mapping | Same (Helm release + test namespace + IAM role + Artifactory mapping) |
| Azure | User-assigned MI, federated credential, test namespace/SA, Helm release, Artifactory OIDC | Helm release, test namespace, MI (+ federated cred), Artifactory OIDC |
| GCP | Google service account (if missing), WIF IAM binding, test namespace/SA, Helm release, Artifactory OIDC | Helm release, test namespace, Artifactory OIDC, Google service account (if this run created it; **not** the WIF pool/provider) |

The Helm release namespace (`jfrog` by default) is uninstalled with Helm but is not explicitly deleted by `oc delete namespace`.

Cleanup uses non-blocking namespace delete (`--wait=false`), timeouts on `az`/`curl`, and progress output. Azure identity delete may warn on timeout — remove manually in Azure if needed.

Fast cleanup (cluster + Helm only; skip cloud identity and Artifactory deletes):

```bash
RESOURCE_GROUP="" ARTIFACTORY_ADMIN_TOKEN="" bash scratch/openshift-test-plan-azure.sh --cleanup
```

## Cloud-specific variables

### AWS

```bash
export AWS_REGION="us-east-1"
# IAM OIDC provider for cluster issuer must exist (ROSA usually has it)
```

### Azure

```bash
export RESOURCE_GROUP="my-aro-rg"
export OIDC_PROVIDER_NAME="openshift-azure-wi"          # same in Helm + Artifactory
export ARTIFACTORY_OIDC_AUDIENCE="*@*"                  # Helm azure_app_client_id (not Entra app ID)
export ARTIFACTORY_USER="jfrog-pull-test"               # Artifactory user in identity mapping
```

**No Entra application registration** for OpenShift projected SAs — only a **user-assigned managed identity** and **federated credential** ([ARO workload identity](https://learn.microsoft.com/en-us/azure/openshift/howto-deploy-configure-application)).

- SA annotation `azure.workload.identity/client-id` = **managed identity client ID**
- Helm `azure_app_client_id` = Artifactory exchange audience (`*@*`), must match identity mapping `token_spec.audience`

`OIDC_PROVIDER_NAME`, `jfrog_oidc_provider_name` in values, and Artifactory OIDC provider name must all match.

## Common issues

| Symptom | Likely cause |
|---------|----------------|
| Artifactory `302` / HTML / `jq` parse error | `ARTIFACTORY_URL` includes `https://` — use hostname only |
| Helm namespace PSA conflict on `audit`/`warn` | Fixed in chart (only `enforce` + `scc.podSecurityLabelSync`); use current `./helm` |
| `namespaces "jfrog" not found` | Create namespace first: `oc create namespace jfrog --dry-run=client -o yaml \| oc apply -f -` (scripts do this in phase 3) |
| `original object Namespace ... not found` | Do **not** use `--create-namespace` with default `openshift.labelNamespacePodSecurity: true` — chart manages the Namespace resource |
| `invalid ownership metadata` on Namespace | Re-run phase 3 (scripts label/annotate `jfrog` for Helm adoption) or `oc label` / `oc annotate` per OpenShift.md |
| Cleanup appears hung | Was `az identity delete` or curl without timeout; current scripts time out and print progress |
| `AccessDenied` on `AssumeRoleWithWebIdentity` | IAM OIDC provider + trust `:sub`/`:aud` vs cluster issuer and workload SA (see OpenShift.md) |
| Phase 1 role `jfrog-pull-app-ns-app-sa` but trust `jfrog-pull-test` | Unset stale `ROLE_NAME` / `APP_NAMESPACE=app-ns` from shell; script derives role name from `APP_NAMESPACE` + `APP_SA_NAME` only |
| Pull test pod exits / wrong command | Scripts use `oc run ... --command -- sleep infinity` |
| Azure pull works but mapping wrong | Align `OIDC_PROVIDER_NAME` across script, values file, and Artifactory |

## Sign-off checklist

- [ ] OpenShift 4.21+
- [ ] `helm template` shows `Platform: openshift` and correct `*-credential-provider.yaml`
- [ ] DaemonSet ready on all workers
- [ ] Merged kubelet config has `tokenAttributes` and correct annotation keys (AWS: `eks.amazonaws.com/role-arn`; Azure: `azure.workload.identity/client-id`)
- [ ] Injector logs show successful merge into platform credential provider config
- [ ] Test pod pulls **without** `imagePullSecrets` (`sleep infinity` pod becomes Ready)
- [ ] Unannotated / wrong SA fails pull when `requireServiceAccount: true`
- [ ] `--cleanup` completes (or WARN only on optional Azure/Artifactory deletes)

## Related files

- [OpenShift.md](../OpenShift.md) — production install and architecture
- [debug.md](../debug.md) — plugin debugging on nodes
- `scratch/results-*.log` — example test run output (not committed as CI artifacts)
