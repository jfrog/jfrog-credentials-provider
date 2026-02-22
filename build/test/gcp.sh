#!/bin/bash
# gcp.sh - E2E tests for GCP (gcp_oidc)
# Expects the following env vars to be set by the caller (GitHub Actions workflow):
#   GKE_CLUSTER_NAME, GCP_PROJECT, GCP_ZONE
#   ARTIFACTORY_URL, MATCH_IMAGES, TEST_IMAGE, HELM_CHART_VERSION
#   GCP_SERVICE_ACCOUNT_EMAIL, GCP_OIDC_AUDIENCE, JFROG_OIDC_PROVIDER_NAME

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/helper.sh"

RUN_ID="${GITHUB_RUN_ID:-$(date +%s)}"

# ---------------------------------------------------------------------------
# gcp_oidc test
# ---------------------------------------------------------------------------
test_gcp_oidc() {
    local ng_name="jfrog-gcp-oidc-${RUN_ID}"
    local release_name="jfrog-cp-gcp-oidc"
    local namespace="jfrog-gcp-oidc"
    local node_label_value="gcp-oidc"
    local values_file="/tmp/values-gcp-oidc.yaml"

    log_step "TEST: GCP OIDC"

    cleanup_gcp() {
        log_step "Cleanup: GCP OIDC"
        cleanup_helm_test "${release_name}" "${namespace}" || true
        delete_node_group_gcp "${GKE_CLUSTER_NAME}" "${GCP_PROJECT}" "${GCP_ZONE}" "${ng_name}" || true
    }
    trap cleanup_gcp EXIT

    create_node_group_gcp \
        "${GKE_CLUSTER_NAME}" \
        "${GCP_PROJECT}" \
        "${GCP_ZONE}" \
        "${ng_name}" \
        "jfrog-test=${node_label_value},credentialsProviderEnabled=true" \
        "${GCP_MACHINE_TYPE:-e2-medium}" \
        "1" \
        "${GCP_SERVICE_ACCOUNT_EMAIL}"

    generate_values "${REPO_ROOT}/examples/gcp-values.yaml" "${values_file}" \
        ".providerConfig[0].artifactoryUrl = \"${ARTIFACTORY_URL}\"" \
        ".providerConfig[0].matchImages[0] = \"${MATCH_IMAGES}\"" \
        ".providerConfig[0].gcp.google_service_account_email = \"${GCP_SERVICE_ACCOUNT_EMAIL}\"" \
        ".providerConfig[0].gcp.jfrog_oidc_audience = \"${GCP_OIDC_AUDIENCE}\"" \
        ".providerConfig[0].gcp.jfrog_oidc_provider_name = \"${JFROG_OIDC_PROVIDER_NAME}\"" \
        ".downloadUrl = \"${DOWNLOAD_URL}\""

    run_helm_test \
        "${release_name}" \
        "${namespace}" \
        "${values_file}" \
        "${TEST_IMAGE}" \
        "jfrog-test" \
        "${node_label_value}" \
        "false" \

    log_info "TEST PASSED: GCP OIDC"

    cleanup_gcp
    trap - EXIT
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    log_step "Starting GCP E2E tests (run: ${RUN_ID})"

    connect_cluster_gcp "${GKE_CLUSTER_NAME}" "${GCP_PROJECT}" "${GCP_ZONE}"

    test_gcp_oidc

    log_step "All GCP E2E tests PASSED"
}

main "$@"
