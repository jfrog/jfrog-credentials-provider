#!/usr/bin/env bash
# OpenShift on Azure — JFrog Kubelet Credential Provider test plan
#
# Entra setup (per Microsoft ARO workload identity): user-assigned managed identity +
# federated credential only — no separate Entra application registration.
#
# Usage:
#   bash scratch/openshift-test-plan-azure.sh              # full test (no teardown)
#   bash scratch/openshift-test-plan-azure.sh --cleanup    # teardown only
#   bash scratch/openshift-test-plan-azure.sh all --cleanup
#   bash scratch/openshift-test-plan-azure.sh --phase 1
#
# Optional: VALUES_FILE=scratch/test-azure.yaml
# See: OpenShift.md, examples/openshift-azure-projected-sa-values.yaml

set -euo pipefail

REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
RUN_CLEANUP=false
FILTERED_ARGS=()
for arg in "$@"; do
  if [[ "$arg" == "--cleanup" ]]; then RUN_CLEANUP=true; else FILTERED_ARGS+=("$arg"); fi
done
set -- "${FILTERED_ARGS[@]}"

# --- Configure these ---
export ARTIFACTORY_URL="${ARTIFACTORY_URL:-your-instance.jfrog.io}"
export ARTIFACTORY_ADMIN_TOKEN="${ARTIFACTORY_ADMIN_TOKEN:-}"
export ARTIFACTORY_USER="${ARTIFACTORY_USER:-jfrog-pull-test}"
export TEST_IMAGE="${TEST_IMAGE:-${ARTIFACTORY_URL}/docker-local/hello-world:latest}"

export HELM_RELEASE="${HELM_RELEASE:-jfrog-cp}"
export HELM_NAMESPACE="${HELM_NAMESPACE:-jfrog}"
export VALUES_FILE="${VALUES_FILE:-${REPO_ROOT}/examples/openshift-azure-projected-sa-values.yaml}"

export APP_NAMESPACE="${APP_NAMESPACE:-jfrog-pull-test}"
export APP_SA_NAME="${APP_SA_NAME:-jfrog-pull-sa}"
# Plugin env azure_app_client_id: Artifactory OIDC exchange audience (not an Entra app ID)
export ARTIFACTORY_OIDC_AUDIENCE="${ARTIFACTORY_OIDC_AUDIENCE:-*@*}"
export OIDC_PROVIDER_NAME="${OIDC_PROVIDER_NAME:-openshift-azure-wi}"
export RESOURCE_GROUP="${RESOURCE_GROUP:-}"
export USER_ASSIGNED_IDENTITY_NAME="${USER_ASSIGNED_IDENTITY_NAME:-jfrog-pull-identity}"
export FEDERATED_IDENTITY_CREDENTIAL_NAME="${FEDERATED_IDENTITY_CREDENTIAL_NAME:-jfrog-pull-federated}"

# Hostname only — no https:// (script adds it). Example: example.jfrog.io
ARTIFACTORY_URL="${ARTIFACTORY_URL#https://}"
ARTIFACTORY_URL="${ARTIFACTORY_URL#http://}"
ARTIFACTORY_URL="${ARTIFACTORY_URL%/}"
export ARTIFACTORY_URL
export ARTIFACTORY_API_BASE="https://${ARTIFACTORY_URL}"

usage() {
  echo "Usage: $0 [all] [--phase N] [--cleanup]"
  echo "  --cleanup   Run teardown (alone, or after 'all')"
}

# OpenShift server version from `oc version` (e.g. 4.21.15) or clusterversion CR.
openshift_server_version() {
  local ver
  ver=$(oc version 2>/dev/null | awk '/^Server Version:/ {print $3; exit}')
  if [[ -z "${ver}" ]]; then
    ver=$(oc get clusterversion version -o jsonpath='{.status.desired.version}' 2>/dev/null | cut -d- -f1)
  fi
  echo "${ver}"
}

# tokenAttributes requires OpenShift 4.21+ (kubelet KubeletServiceAccountTokenForCredentialProviders).
openshift_meets_token_attributes_version() {
  local ver="${1:-$(openshift_server_version)}"
  local major minor
  [[ -z "${ver}" ]] && return 1
  major="${ver%%.*}"
  minor="${ver#*.}"
  minor="${minor%%.*}"
  [[ "${major}" -gt 4 ]] && return 0
  [[ "${major}" -eq 4 && "${minor}" -ge 21 ]] && return 0
  return 1
}

# Wall-clock limit for slow CLI calls (no GNU timeout required).
run_with_timeout() {
  local secs=$1
  shift
  "$@" &
  local pid=$!
  local waited=0
  while kill -0 "${pid}" 2>/dev/null && [[ ${waited} -lt ${secs} ]]; do
    sleep 1
    waited=$((waited + 1))
  done
  if kill -0 "${pid}" 2>/dev/null; then
    kill "${pid}" 2>/dev/null
    wait "${pid}" 2>/dev/null || true
    echo "WARN: timed out after ${secs}s: $*" >&2
    return 124
  fi
  wait "${pid}"
}

CURL_OPTS=(--connect-timeout 10 --max-time 30)

# POST/GET Artifactory Access API; fails clearly on redirects (common when URL has https:// prefix).
# Optional: --allow-404 (treat 404 as success, for idempotent DELETE).
artifactory_api() {
  local method=$1 path=$2
  shift 2
  local allow_404=false
  if [[ "${1:-}" == "--allow-404" ]]; then
    allow_404=true
    shift
  fi
  local url="${ARTIFACTORY_API_BASE}${path}"
  local body_file http_code
  body_file="$(mktemp)"
  http_code="$(curl -sS "${CURL_OPTS[@]}" -X "${method}" "${url}" -o "${body_file}" -w "%{http_code}" \
    -H "Authorization: Bearer ${ARTIFACTORY_ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    "$@")"
  if [[ "${http_code}" =~ ^(301|302|303|307|308)$ ]]; then
    echo "ERROR: HTTP ${http_code} redirect from ${url}" >&2
    echo "  Set ARTIFACTORY_URL to hostname only (e.g. example.jfrog.io), not https://..." >&2
    head -20 "${body_file}" >&2
    rm -f "${body_file}"
    return 1
  fi
  if [[ "${allow_404}" == true && "${http_code}" == "404" ]]; then
    rm -f "${body_file}"
    return 0
  fi
  if [[ "${http_code}" -ge 400 ]]; then
    echo "ERROR: HTTP ${http_code} from ${method} ${path}" >&2
    cat "${body_file}" >&2
    rm -f "${body_file}"
    return 1
  fi
  cat "${body_file}"
  rm -f "${body_file}"
}

# Release metadata lives in this namespace. The chart's openshift-namespace.yaml applies
# Pod Security labels — do not pass --create-namespace (conflicts with chart Namespace SSA).
ensure_helm_release_namespace() {
  oc create namespace "${HELM_NAMESPACE}" --dry-run=client -o yaml | oc apply -f - >/dev/null
  oc label namespace "${HELM_NAMESPACE}" \
    app.kubernetes.io/managed-by=Helm \
    --overwrite
  oc annotate namespace "${HELM_NAMESPACE}" \
    "meta.helm.sh/release-name=${HELM_RELEASE}" \
    "meta.helm.sh/release-namespace=${HELM_NAMESPACE}" \
    --overwrite
  echo "OK: release namespace ${HELM_NAMESPACE} exists (Helm adoption metadata applied)"
}

# ---------------------------------------------------------------------------
phase_cleanup() {
  echo "=== Cleanup: removing test resources ==="

  echo "→ Helm uninstall ${HELM_RELEASE} in ${HELM_NAMESPACE}..."
  helm uninstall "${HELM_RELEASE}" -n "${HELM_NAMESPACE}" --ignore-not-found 2>/dev/null || true

  echo "→ Workload namespace ${APP_NAMESPACE}..."
  if oc get namespace "${APP_NAMESPACE}" &>/dev/null; then
    oc delete namespace "${APP_NAMESPACE}" --wait=false --ignore-not-found
    echo "  delete requested (not waiting for termination)"
  else
    echo "  already gone"
  fi

  if [[ -n "${RESOURCE_GROUP}" ]]; then
    echo "→ Azure managed identity ${USER_ASSIGNED_IDENTITY_NAME}..."
    if run_with_timeout 30 az identity show \
      --name "${USER_ASSIGNED_IDENTITY_NAME}" \
      --resource-group "${RESOURCE_GROUP}" >/dev/null 2>&1; then
      run_with_timeout 60 az identity federated-credential delete \
        --name "${FEDERATED_IDENTITY_CREDENTIAL_NAME}" \
        --identity-name "${USER_ASSIGNED_IDENTITY_NAME}" \
        --resource-group "${RESOURCE_GROUP}" 2>/dev/null || true
      if run_with_timeout 90 az identity delete \
        --name "${USER_ASSIGNED_IDENTITY_NAME}" \
        --resource-group "${RESOURCE_GROUP}"; then
        echo "  deleted"
      else
        echo "  WARN: delete timed out or failed — remove manually in Azure if needed"
      fi
    else
      echo "  not found (skip)"
    fi
  fi

  if [[ -n "${ARTIFACTORY_ADMIN_TOKEN:-}" ]]; then
    echo "→ Artifactory OIDC ${OIDC_PROVIDER_NAME}..."
    curl -sf "${CURL_OPTS[@]}" -X DELETE \
      "${ARTIFACTORY_API_BASE}/access/api/v1/oidc/${OIDC_PROVIDER_NAME}/identity_mappings/${OIDC_PROVIDER_NAME}-mapping" \
      -H "Authorization: Bearer ${ARTIFACTORY_ADMIN_TOKEN}" 2>/dev/null || true
    if curl -sf "${CURL_OPTS[@]}" -X DELETE \
      "${ARTIFACTORY_API_BASE}/access/api/v1/oidc/${OIDC_PROVIDER_NAME}" \
      -H "Authorization: Bearer ${ARTIFACTORY_ADMIN_TOKEN}" 2>/dev/null; then
      echo "  removed"
    else
      echo "  WARN: delete failed or timed out (remove manually if needed)"
    fi
  fi

  echo "Cleanup complete"
}

phase0_prerequisites() {
  echo "=== Phase 0: Prerequisites ==="
  oc version
  OCP_SERVER_VERSION="$(openshift_server_version)"
  if openshift_meets_token_attributes_version "${OCP_SERVER_VERSION}"; then
    echo "OK: OpenShift ${OCP_SERVER_VERSION} meets 4.21+ requirement for tokenAttributes"
  else
    echo "WARN: cluster should be OpenShift 4.21+ for tokenAttributes (detected: ${OCP_SERVER_VERSION:-unknown})"
  fi

  export ARO_OIDC_ISSUER="$(oc get authentication cluster -o jsonpath='{.spec.serviceAccountIssuer}')"
  echo "Cluster OIDC issuer: ${ARO_OIDC_ISSUER}"

  oc describe deployment pod-identity-webhook -n openshift-cloud-credential-operator \
    | grep 'target.workload.openshift.io/management' \
    && echo "OK: pod-identity-webhook managed" \
    || echo "WARN: verify pod-identity-webhook per OpenShift.md"
}

phase1_entra_workload_identity() {
  echo "=== Phase 1: User-assigned managed identity + federated credential + SA ==="
  export ARO_OIDC_ISSUER="${ARO_OIDC_ISSUER:-$(oc get authentication cluster -o jsonpath='{.spec.serviceAccountIssuer}')}"
  test -n "${RESOURCE_GROUP}" || { echo "Set RESOURCE_GROUP"; exit 1; }

  az identity create \
    --name "${USER_ASSIGNED_IDENTITY_NAME}" \
    --resource-group "${RESOURCE_GROUP}" \
    --output none 2>/dev/null || true

  export USER_ASSIGNED_IDENTITY_CLIENT_ID="$(az identity show \
    --name "${USER_ASSIGNED_IDENTITY_NAME}" \
    --resource-group "${RESOURCE_GROUP}" \
    --query clientId -o tsv)"
  echo "USER_ASSIGNED_IDENTITY_CLIENT_ID=${USER_ASSIGNED_IDENTITY_CLIENT_ID}"

  az identity federated-credential create \
    --name "${FEDERATED_IDENTITY_CREDENTIAL_NAME}" \
    --identity-name "${USER_ASSIGNED_IDENTITY_NAME}" \
    --resource-group "${RESOURCE_GROUP}" \
    --issuer "${ARO_OIDC_ISSUER}" \
    --subject "system:serviceaccount:${APP_NAMESPACE}:${APP_SA_NAME}" \
    --audience "api://AzureADTokenExchange" \
    --output none 2>/dev/null || echo "Federated credential may already exist"

  oc create namespace "${APP_NAMESPACE}" --dry-run=client -o yaml | oc apply -f -
  oc create serviceaccount "${APP_SA_NAME}" -n "${APP_NAMESPACE}" --dry-run=client -o yaml | oc apply -f -
  oc annotate serviceaccount "${APP_SA_NAME}" -n "${APP_NAMESPACE}" \
    azure.workload.identity/client-id="${USER_ASSIGNED_IDENTITY_CLIENT_ID}" \
    JFrogExchange=true --overwrite
  oc get sa "${APP_SA_NAME}" -n "${APP_NAMESPACE}" -o yaml | grep -E 'client-id|JFrogExchange'
}

phase2_artifactory_oidc() {
  echo "=== Phase 2: Artifactory OIDC provider + identity mapping ==="
  export ARO_OIDC_ISSUER="${ARO_OIDC_ISSUER:-$(oc get authentication cluster -o jsonpath='{.spec.serviceAccountIssuer}')}"
  test -n "${ARTIFACTORY_ADMIN_TOKEN}" || { echo "ERROR: Set ARTIFACTORY_ADMIN_TOKEN" >&2; exit 1; }
  echo "Artifactory API base: ${ARTIFACTORY_API_BASE}"

  artifactory_api DELETE "/access/api/v1/oidc/${OIDC_PROVIDER_NAME}/identity_mappings/${OIDC_PROVIDER_NAME}-mapping" --allow-404
  artifactory_api DELETE "/access/api/v1/oidc/${OIDC_PROVIDER_NAME}" --allow-404

  artifactory_api POST "/access/api/v1/oidc" \
    -d "{
      \"name\": \"${OIDC_PROVIDER_NAME}\",
      \"issuer_url\": \"${ARO_OIDC_ISSUER}\",
      \"provider_type\": \"Azure\",
      \"token_issuer\": \"${ARO_OIDC_ISSUER}\",
      \"use_default_proxy\": false,
      \"description\": \"OpenShift on Azure workload identity\"
    }"
  echo "OK: OIDC provider"

  artifactory_api POST "/access/api/v1/oidc/${OIDC_PROVIDER_NAME}/identity_mappings" \
    -d "{
      \"name\": \"${OIDC_PROVIDER_NAME}-mapping\",
      \"description\": \"OpenShift workload identity mapping\",
      \"claims\": {
        \"aud\": \"api://AzureADTokenExchange\",
        \"iss\": \"${ARO_OIDC_ISSUER}\",
        \"sub\": \"system:serviceaccount:${APP_NAMESPACE}:${APP_SA_NAME}\"
      },
      \"token_spec\": {
        \"username\": \"${ARTIFACTORY_USER}\",
        \"scope\": \"applied-permissions/user\",
        \"audience\": \"${ARTIFACTORY_OIDC_AUDIENCE}\",
        \"expires_in\": 18000
      },
      \"priority\": 1
    }"
  echo "OK: identity mapping"

  artifactory_api GET "/access/api/v1/oidc/${OIDC_PROVIDER_NAME}" | jq .
}

phase3_helm_install() {
  echo "=== Phase 3: Helm install (local chart) ==="
  ensure_helm_release_namespace

  helm template "${HELM_RELEASE}" "${REPO_ROOT}/helm" -f "${VALUES_FILE}" \
    --set "providerConfig[0].artifactoryUrl=${ARTIFACTORY_URL}" \
    --set "providerConfig[0].azure.azure_app_client_id=${ARTIFACTORY_OIDC_AUDIENCE}" \
    --set "providerConfig[0].azure.jfrog_oidc_provider_name=${OIDC_PROVIDER_NAME}" \
    | grep -E 'Platform: openshift|acr-credential-provider' | head -5

  helm upgrade --install "${HELM_RELEASE}" "${REPO_ROOT}/helm" \
    --namespace "${HELM_NAMESPACE}" \
    -f "${VALUES_FILE}" \
    --set "providerConfig[0].artifactoryUrl=${ARTIFACTORY_URL}" \
    --set "providerConfig[0].azure.azure_app_client_id=${ARTIFACTORY_OIDC_AUDIENCE}" \
    --set "providerConfig[0].azure.jfrog_oidc_provider_name=${OIDC_PROVIDER_NAME}"

  oc rollout status daemonset -n "${HELM_NAMESPACE}" -l app.kubernetes.io/name=jfrog-credential-provider --timeout=300s
}

phase4_verify_chart() {
  echo "=== Phase 4: Verify chart / namespace / SCC ==="
  oc get namespace "${HELM_NAMESPACE}" --show-labels | grep 'pod-security.kubernetes.io/enforce=privileged'
  oc get pods -n "${HELM_NAMESPACE}" -l app.kubernetes.io/name=jfrog-credential-provider
}

phase5_verify_node() {
  echo "=== Phase 5: Verify node installation ==="
  WORKER="$(oc get nodes -l node-role.kubernetes.io/worker= -o jsonpath='{.items[0].metadata.name}')"

  oc debug "node/${WORKER}" -- chroot /host bash -c '
    set -e
    ls -la /usr/libexec/kubelet-image-credential-provider-plugins/jfrog-credentials-provider
    grep -q jfrog-credentials-provider /etc/kubernetes/credential-providers/acr-credential-provider.yaml
    grep -q azure.workload.identity/client-id /etc/kubernetes/credential-providers/acr-credential-provider.yaml
    grep -q tokenAttributes /etc/kubernetes/credential-providers/acr-credential-provider.yaml
    echo OK: plugin and merged acr-credential-provider.yaml
  '
}

phase6_injector_logs() {
  echo "=== Phase 6: Injector logs ==="
  POD="$(oc get pods -n "${HELM_NAMESPACE}" -l app.kubernetes.io/name=jfrog-credential-provider -o jsonpath='{.items[0].metadata.name}')"
  oc logs -n "${HELM_NAMESPACE}" "${POD}" -c jfrog-credential-provider-injector | tail -40
}

phase7_positive_pull() {
  echo "=== Phase 7: Positive test — pull without imagePullSecrets ==="
  oc delete pod jfrog-pull-test -n "${APP_NAMESPACE}" --ignore-not-found
  oc run jfrog-pull-test -n "${APP_NAMESPACE}" \
    --image="${TEST_IMAGE}" \
    --overrides="{\"spec\":{\"serviceAccountName\":\"${APP_SA_NAME}\"}}" \
    --restart=Never \
    --command -- sleep infinity
  oc wait --for=condition=Ready pod/jfrog-pull-test -n "${APP_NAMESPACE}" --timeout=120s
  echo "OK: image pull succeeded"
}

phase8_negative_tests() {
  echo "=== Phase 8: Negative test — pod without WI annotations ==="
  oc create serviceaccount default-pull -n "${APP_NAMESPACE}"
  oc delete pod jfrog-pull-test-neg -n "${APP_NAMESPACE}" --ignore-not-found
  oc run jfrog-pull-test-neg -n "${APP_NAMESPACE}" \
    --image="${TEST_IMAGE}" \
    --overrides='{"spec":{"serviceAccountName":"default-pull"}}' \
    --restart=Never \
    --command -- sleep infinity
  sleep 15
  oc get pod jfrog-pull-test-neg -n "${APP_NAMESPACE}" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' ; echo
  echo "(expect ErrImagePull or ImagePullBackOff)"
}

run_phase() {
  case "$1" in
    0) phase0_prerequisites ;;
    1) phase1_entra_workload_identity ;;
    2) phase2_artifactory_oidc ;;
    3) phase3_helm_install ;;
    4) phase4_verify_chart ;;
    5) phase5_verify_node ;;
    6) phase6_injector_logs ;;
    7) phase7_positive_pull ;;
    8) phase8_negative_tests ;;
    *) echo "Unknown phase $1"; exit 1 ;;
  esac
}

if [[ $# -eq 0 && "$RUN_CLEANUP" == true ]]; then
  phase_cleanup
  exit 0
fi

if [[ $# -eq 0 || "${1:-}" == "all" ]]; then
  for i in 0 1 2 3 4 5 6 7 8; do run_phase "$i"; done
  echo "=== Done ==="
  [[ "$RUN_CLEANUP" == true ]] && phase_cleanup
elif [[ "${1:-}" == "--phase" ]]; then
  run_phase "${2:?phase number}"
  [[ "$RUN_CLEANUP" == true ]] && phase_cleanup
else
  usage
  exit 1
fi
