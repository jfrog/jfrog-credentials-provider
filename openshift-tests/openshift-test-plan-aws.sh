#!/usr/bin/env bash
# OpenShift on AWS — JFrog Kubelet Credential Provider test plan
#
# Usage:
#   bash scratch/openshift-test-plan-aws.sh              # full test (no teardown)
#   bash scratch/openshift-test-plan-aws.sh --cleanup    # teardown only
#   bash scratch/openshift-test-plan-aws.sh all --cleanup # full test, then teardown
#   bash scratch/openshift-test-plan-aws.sh --phase 3    # single phase
#
# See: OpenShift.md, examples/openshift-aws-projected-sa-values.yaml

set -euo pipefail

# Avoid aws cli paging into less during non-interactive test runs.
export AWS_PAGER=""
export AWS_CLI_AUTO_PROMPT=off

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
export VALUES_FILE="${VALUES_FILE:-${REPO_ROOT}/examples/openshift-aws-projected-sa-values.yaml}"

export APP_NAMESPACE="${APP_NAMESPACE:-jfrog-pull-test}"
export APP_SA_NAME="${APP_SA_NAME:-jfrog-pull-sa}"
export AWS_REGION="${AWS_REGION:-us-east-1}"

# Always derived from APP_NAMESPACE + APP_SA_NAME (ignore a stale ROLE_NAME from the shell or OpenShift.md examples).
resolve_role_name() {
  ROLE_NAME="jfrog-pull-${APP_NAMESPACE}-${APP_SA_NAME}"
  export ROLE_NAME
}
resolve_role_name

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

CURL_OPTS=(--connect-timeout 10 --max-time 30)

# True when Artifactory indicates no existing resource (idempotent DELETE).
artifactory_response_is_missing() {
  local http_code=$1 body_file=$2
  [[ "${http_code}" == "404" ]] && return 0
  if [[ "${http_code}" == "400" ]] && grep -q 'Could not find user' "${body_file}"; then
    return 0
  fi
  return 1
}

# POST/GET/PUT/DELETE Artifactory Access API; prints HTTP status and body on failure.
# Optional: --allow-missing (404, or 400 "Could not find user" on DELETE — Artifactory has no binding yet).
artifactory_api() {
  local method=$1 path=$2
  shift 2
  local allow_missing=false
  if [[ "${1:-}" == "--allow-missing" || "${1:-}" == "--allow-404" ]]; then
    allow_missing=true
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
  if [[ "${allow_missing}" == true ]] && artifactory_response_is_missing "${http_code}" "${body_file}"; then
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

# Returns the IAM role ARN mapped to a user, or empty string if none.
artifactory_get_iam_role_arn_for_user() {
  local user=$1
  local body
  body="$(artifactory_api GET "/access/api/v1/aws/iam_role/${user}" --allow-missing 2>/dev/null || true)"
  [[ -z "${body}" ]] && return 0
  jq -r '.iam_role // empty' <<< "${body}" 2>/dev/null || true
}

# Artifactory allows one user per IAM role ARN. Remove the role from any other user before PUT.
artifactory_unassign_iam_role_from_other_users() {
  local role_arn=$1 keep_user=$2
  local users_body user existing_role

  if ! command -v jq >/dev/null 2>&1; then
    echo "WARN: jq is required to scan for conflicting IAM role mappings" >&2
    return 0
  fi

  users_body="$(artifactory_api GET "/access/api/v1/users?limit=1000" 2>/dev/null || true)"
  if [[ -z "${users_body}" ]]; then
    echo "WARN: could not list Artifactory users; if PUT returns 409, unmap ${role_arn} from the other user manually" >&2
    return 0
  fi

  while IFS= read -r user; do
    [[ -z "${user}" || "${user}" == "${keep_user}" ]] && continue
    existing_role="$(artifactory_get_iam_role_arn_for_user "${user}")"
    if [[ "${existing_role}" == "${role_arn}" ]]; then
      echo "Unassigning ${role_arn} from Artifactory user ${user} (one role ARN per user in Artifactory)"
      artifactory_api DELETE "/access/api/v1/aws/iam_role/${user}" --allow-missing
    fi
  done < <(jq -r 'if type == "array" then .[] else . end | .username? // empty' <<< "${users_body}")

  if [[ -n "${ARTIFACTORY_IAM_ROLE_CONFLICT_USERS:-}" ]]; then
    local extra
    IFS=',' read -ra extras <<< "${ARTIFACTORY_IAM_ROLE_CONFLICT_USERS}"
    for extra in "${extras[@]}"; do
      extra="${extra#"${extra%%[![:space:]]*}"}"
      extra="${extra%"${extra##*[![:space:]]}"}"
      [[ -z "${extra}" || "${extra}" == "${keep_user}" ]] && continue
      existing_role="$(artifactory_get_iam_role_arn_for_user "${extra}")"
      if [[ "${existing_role}" == "${role_arn}" ]]; then
        echo "Unassigning ${role_arn} from Artifactory user ${extra} (ARTIFACTORY_IAM_ROLE_CONFLICT_USERS)"
        artifactory_api DELETE "/access/api/v1/aws/iam_role/${extra}" --allow-missing
      fi
    done
  fi
}

# Release metadata lives in this namespace. The chart's openshift-namespace.yaml applies
# Pod Security labels — do not pass --create-namespace (conflicts with chart Namespace SSA).
ensure_helm_release_namespace() {
  # Helm release secrets need the namespace to exist. The chart also manages Namespace
  # (Pod Security labels). Pre-create with Helm ownership metadata so install can adopt it
  # (do not use --create-namespace — conflicts with chart Namespace resource).
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
  helm uninstall "${HELM_RELEASE}" -n "${HELM_NAMESPACE}" --ignore-not-found 2>/dev/null || true
  if oc get namespace "${APP_NAMESPACE}" &>/dev/null; then
    oc delete namespace "${APP_NAMESPACE}" --wait=false --ignore-not-found
    echo "Delete requested for ${APP_NAMESPACE} (not waiting for termination)"
  else
    echo "Namespace ${APP_NAMESPACE} already deleted"
  fi

  if aws iam get-role --role-name "${ROLE_NAME}" >/dev/null 2>&1; then
    aws iam delete-role --role-name "${ROLE_NAME}" && echo "Deleted IAM role ${ROLE_NAME}"
  fi

  if [[ -n "${ARTIFACTORY_ADMIN_TOKEN:-}" ]]; then
    if artifactory_api DELETE "/access/api/v1/aws/iam_role/${ARTIFACTORY_USER}" --allow-missing; then
      echo "Removed Artifactory IAM role mapping for ${ARTIFACTORY_USER} (or none existed)"
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

  export OIDC_ISSUER="$(oc get authentication cluster -o jsonpath='{.spec.serviceAccountIssuer}')"
  export OIDC_HOSTPATH="${OIDC_ISSUER#https://}"
  export AWS_ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
  echo "OIDC issuer: ${OIDC_ISSUER}"
  echo "AWS account: ${AWS_ACCOUNT_ID}"

  if aws iam list-open-id-connect-providers 2>/dev/null | grep -q "${OIDC_HOSTPATH}"; then
    echo "OK: IAM OIDC provider exists for cluster issuer"
  else
    echo "WARN: self-managed OpenShift on AWS may need an IAM OIDC provider for ${OIDC_HOSTPATH}"
  fi
}

phase1_workload_identity() {
  echo "=== Phase 1: Workload IAM role + ServiceAccount ==="
  resolve_role_name
  echo "APP_NAMESPACE=${APP_NAMESPACE} APP_SA_NAME=${APP_SA_NAME} ROLE_NAME=${ROLE_NAME}"

  oc create namespace "${APP_NAMESPACE}" --dry-run=client -o yaml | oc apply -f -

  export OIDC_ISSUER="${OIDC_ISSUER:-$(oc get authentication cluster -o jsonpath='{.spec.serviceAccountIssuer}')}"
  export OIDC_HOSTPATH="${OIDC_HOSTPATH:-${OIDC_ISSUER#https://}}"
  export AWS_ACCOUNT_ID="${AWS_ACCOUNT_ID:-$(aws sts get-caller-identity --query Account --output text)}"

  cat > /tmp/jfrog-trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_HOSTPATH}"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "${OIDC_HOSTPATH}:sub": "system:serviceaccount:${APP_NAMESPACE}:${APP_SA_NAME}",
        "${OIDC_HOSTPATH}:aud": "sts.amazonaws.com"
      }
    }
  }]
}
EOF

  if aws iam get-role --role-name "${ROLE_NAME}" >/dev/null 2>&1; then
    echo "IAM role ${ROLE_NAME} already exists (trust policy is not updated; delete the role to recreate)"
  else
    aws iam create-role \
      --role-name "${ROLE_NAME}" \
      --assume-role-policy-document file:///tmp/jfrog-trust-policy.json \
      --output text --query 'Role.Arn'
    echo
  fi
  export ROLE_ARN="$(aws iam get-role --role-name "${ROLE_NAME}" --query 'Role.Arn' --output text)"
  echo "ROLE_ARN=${ROLE_ARN}"

  oc create serviceaccount "${APP_SA_NAME}" -n "${APP_NAMESPACE}" --dry-run=client -o yaml | oc apply -f -
  oc annotate serviceaccount "${APP_SA_NAME}" -n "${APP_NAMESPACE}" \
    eks.amazonaws.com/role-arn="${ROLE_ARN}" \
    JFrogExchange=true --overwrite
  oc get sa "${APP_SA_NAME}" -n "${APP_NAMESPACE}" -o yaml | grep -E 'role-arn|JFrogExchange'
}

phase2_artifactory() {
  echo "=== Phase 2: Artifactory IAM role mapping ==="
  test -n "${ARTIFACTORY_ADMIN_TOKEN}" || { echo "ERROR: Set ARTIFACTORY_ADMIN_TOKEN" >&2; exit 1; }
  echo "Artifactory API base: ${ARTIFACTORY_API_BASE}"

  if [[ -z "${ROLE_ARN:-}" ]]; then
    if aws iam get-role --role-name "${ROLE_NAME}" >/dev/null 2>&1; then
      export ROLE_ARN="$(aws iam get-role --role-name "${ROLE_NAME}" --query 'Role.Arn' --output text)"
      echo "ROLE_ARN=${ROLE_ARN} (from existing IAM role ${ROLE_NAME})"
    else
      echo "ERROR: ROLE_ARN is not set; run phase 1 first or export ROLE_ARN" >&2
      exit 1
    fi
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: jq is required for phase 2 (brew install jq / apt install jq)" >&2
    exit 1
  fi

  existing_role="$(artifactory_get_iam_role_arn_for_user "${ARTIFACTORY_USER}")"
  if [[ "${existing_role}" == "${ROLE_ARN}" ]]; then
    echo "OK: ${ARTIFACTORY_USER} is already mapped to ${ROLE_ARN}"
    artifactory_api GET "/access/api/v1/aws/iam_role/${ARTIFACTORY_USER}" | jq .
    echo "OK: Artifactory IAM role mapping verified"
    return 0
  fi

  artifactory_unassign_iam_role_from_other_users "${ROLE_ARN}" "${ARTIFACTORY_USER}"

  if [[ -n "${existing_role}" && "${existing_role}" != "${ROLE_ARN}" ]]; then
    artifactory_api DELETE "/access/api/v1/aws/iam_role/${ARTIFACTORY_USER}" --allow-missing
    echo "OK: removed prior mapping for ${ARTIFACTORY_USER} (${existing_role})"
  else
    echo "OK: no prior IAM role binding for ${ARTIFACTORY_USER}"
  fi

  echo "Creating IAM role mapping for Artifactory user: ${ARTIFACTORY_USER}"
  echo "  (user must already exist in Artifactory — create in UI if needed)"
  artifactory_api PUT "/access/api/v1/aws/iam_role" \
    -d "{\"username\": \"${ARTIFACTORY_USER}\", \"iam_role\": \"${ROLE_ARN}\"}"
  echo "OK: created IAM role binding"

  artifactory_api GET "/access/api/v1/aws/iam_role/${ARTIFACTORY_USER}" | jq .
  echo "OK: Artifactory IAM role mapping verified"
}

phase3_helm_install() {
  echo "=== Phase 3: Helm install (local chart) ==="
  ensure_helm_release_namespace

  helm template "${HELM_RELEASE}" "${REPO_ROOT}/helm" -f "${VALUES_FILE}" \
    --set "providerConfig[0].artifactoryUrl=${ARTIFACTORY_URL}" \
    --set "providerConfig[0].aws.aws_region=${AWS_REGION}" \
    | grep -E 'Platform: openshift|ecr-credential-provider' | head -5

  helm upgrade --install "${HELM_RELEASE}" "${REPO_ROOT}/helm" \
    --namespace "${HELM_NAMESPACE}" \
    -f "${VALUES_FILE}" \
    --set "providerConfig[0].artifactoryUrl=${ARTIFACTORY_URL}" \
    --set "providerConfig[0].aws.aws_region=${AWS_REGION}"

  oc rollout status daemonset -n "${HELM_NAMESPACE}" -l app.kubernetes.io/name=jfrog-credential-provider --timeout=300s
}

phase4_verify_chart() {
  echo "=== Phase 4: Verify chart / namespace / SCC ==="
  oc get namespace "${HELM_NAMESPACE}" --show-labels | grep 'pod-security.kubernetes.io/enforce=privileged' \
    && echo "OK: namespace Pod Security labels" \
    || echo "FAIL: missing privileged Pod Security labels on ${HELM_NAMESPACE}"

  oc get rolebinding -n "${HELM_NAMESPACE}" | grep -i scc-privileged || echo "WARN: check openshift.grantPrivilegedSCC"

  READY="$(oc get daemonset -n "${HELM_NAMESPACE}" -l app.kubernetes.io/name=jfrog-credential-provider -o jsonpath='{.items[0].status.numberReady}')"
  DESIRED="$(oc get daemonset -n "${HELM_NAMESPACE}" -l app.kubernetes.io/name=jfrog-credential-provider -o jsonpath='{.items[0].status.desiredNumberScheduled}')"
  test "${READY}" = "${DESIRED}" && echo "OK: DaemonSet ${READY}/${DESIRED} ready"
}

phase5_verify_node() {
  echo "=== Phase 5: Verify node installation ==="
  WORKER="$(oc get nodes -l node-role.kubernetes.io/worker= -o jsonpath='{.items[0].metadata.name}')"
  echo "Checking worker: ${WORKER}"

  oc debug "node/${WORKER}" -- chroot /host bash -c '
    set -e
    ls -la /usr/libexec/kubelet-image-credential-provider-plugins/jfrog-credentials-provider
    grep -q jfrog-credentials-provider /etc/kubernetes/credential-providers/ecr-credential-provider.yaml
    grep -q tokenAttributes /etc/kubernetes/credential-providers/ecr-credential-provider.yaml
    grep -q eks.amazonaws.com/role-arn /etc/kubernetes/credential-providers/ecr-credential-provider.yaml
    echo OK: plugin binary and merged ecr-credential-provider.yaml
    echo "--- host log (last 20 lines) ---"
    tail -20 /var/log/jfrog-credentials-provider/jfrog-credentials-provider.log 2>/dev/null || echo "(no log file yet)"
  '
}

phase6_injector_logs() {
  echo "=== Phase 6: Injector init container logs ==="
  POD="$(oc get pods -n "${HELM_NAMESPACE}" -l app.kubernetes.io/name=jfrog-credential-provider -o jsonpath='{.items[0].metadata.name}')"
  oc logs -n "${HELM_NAMESPACE}" "${POD}" -c jfrog-credential-provider-injector | tail -40
  echo "--- expect: Platform: openshift, bind-mount, merge success, NOT /etc/eks/ ---"
}

phase7_positive_pull() {
  echo "=== Phase 7: Positive test — pull without imagePullSecrets ==="
  oc delete pod jfrog-pull-test -n "${APP_NAMESPACE}" --ignore-not-found
  oc run jfrog-pull-test -n "${APP_NAMESPACE}" \
    --image="${TEST_IMAGE}" \
    --overrides="{\"spec\":{\"serviceAccountName\":\"${APP_SA_NAME}\"}}" \
    --restart=Never \
    --command -- sleep infinity
  sleep 5
  oc wait --for=condition=Ready pod/jfrog-pull-test -n "${APP_NAMESPACE}" --timeout=120s \
    && echo "OK: pod pulled image successfully" \
    || { oc describe pod jfrog-pull-test -n "${APP_NAMESPACE}"; exit 1; }
}

phase8_negative_tests() {
  echo "=== Phase 8: Negative test — pod without IRSA annotations ==="
  oc create serviceaccount no-irsa -n "${APP_NAMESPACE}"
  oc delete pod jfrog-pull-test-neg -n "${APP_NAMESPACE}" --ignore-not-found
  oc run jfrog-pull-test-neg -n "${APP_NAMESPACE}" \
    --image="${TEST_IMAGE}" \
    --overrides='{"spec":{"serviceAccountName":"no-irsa"}}' \
    --restart=Never \
    --command -- sleep infinity
  sleep 15
  oc get pod jfrog-pull-test-neg -n "${APP_NAMESPACE}" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' ; echo
  echo "   (expect ErrImagePull or ImagePullBackOff when requireServiceAccount is true)"
}

run_phase() {
  case "$1" in
    0) phase0_prerequisites ;;
    1) phase1_workload_identity ;;
    2) phase2_artifactory ;;
    3) phase3_helm_install ;;
    4) phase4_verify_chart ;;
    5) phase5_verify_node ;;
    6) phase6_injector_logs ;;
    7) phase7_positive_pull ;;
    8) phase8_negative_tests ;;
    *) echo "Unknown phase $1"; exit 1 ;;
  esac
}

# --- main ---
if [[ $# -eq 0 && "$RUN_CLEANUP" == true ]]; then
  phase_cleanup
  exit 0
fi

if [[ $# -eq 0 || "${1:-}" == "all" ]]; then
  phase0_prerequisites
  phase1_workload_identity
  phase2_artifactory
  phase3_helm_install
  phase4_verify_chart
  phase5_verify_node
  phase6_injector_logs
  phase7_positive_pull
  phase8_negative_tests
  echo "=== Done ==="
  [[ "$RUN_CLEANUP" == true ]] && phase_cleanup
elif [[ "${1:-}" == "--phase" ]]; then
  run_phase "${2:?phase number}"
  [[ "$RUN_CLEANUP" == true ]] && phase_cleanup
else
  usage
  exit 1
fi
