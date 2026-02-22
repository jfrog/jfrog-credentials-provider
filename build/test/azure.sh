#!/bin/bash
# azure.sh - E2E tests for Azure (azure_oidc)
# Expects the following env vars to be set by the caller (GitHub Actions workflow):
#   AKS_CLUSTER_NAME, AKS_RESOURCE_GROUP
#   ARTIFACTORY_URL, MATCH_IMAGES, TEST_IMAGE
#   AZURE_APP_CLIENT_ID, AZURE_TENANT_ID, AZURE_NODEPOOL_CLIENT_ID,
#   JFROG_OIDC_PROVIDER_NAME

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/helper.sh"

RUN_ID="${GITHUB_RUN_ID:-$(date +%s)}"

# ---------------------------------------------------------------------------
# azure_oidc test
# ---------------------------------------------------------------------------
test_azure_oidc() {
    local ng_name="jfrogazoidc${RUN_ID}"
    # Azure node pool names must be <= 12 chars alphanumeric; truncate RUN_ID
    ng_name="jfaz${RUN_ID: -8}"
    local release_name="jfrog-cp-azure-oidc"
    local namespace="jfrog-azure-oidc"
    local node_label_value="azure-oidc"
    local values_file="/tmp/values-azure-oidc.yaml"

    log_step "TEST: Azure OIDC"

    cleanup_azure() {
        log_step "Cleanup: Azure OIDC"
        cleanup_helm_test "${release_name}" "${namespace}" || true
        delete_node_group_azure "${AKS_CLUSTER_NAME}" "${AKS_RESOURCE_GROUP}" "${ng_name}" || true
    }
    trap cleanup_azure EXIT

    create_node_group_azure \
        "${AKS_CLUSTER_NAME}" \
        "${AKS_RESOURCE_GROUP}" \
        "${ng_name}" \
        "jfrog-test=${node_label_value},credentialsProviderEnabled=true" \
        "${AZURE_NODE_VM_SIZE:-Standard_D2pds_v5}" \
        "${AZURE_NODE_COUNT:-1}" \
        "${AZURE_NODEPOOL_CLIENT_ID}"

    generate_values "${REPO_ROOT}/examples/azure-values.yaml" "${values_file}" \
        ".providerConfig[0].artifactoryUrl = \"${ARTIFACTORY_URL}\"" \
        ".providerConfig[0].matchImages[0] = \"${MATCH_IMAGES}\"" \
        ".providerConfig[0].azure.azure_app_client_id = \"${AZURE_APP_CLIENT_ID}\"" \
        ".providerConfig[0].azure.azure_tenant_id = \"${AZURE_TENANT_ID}\"" \
        ".providerConfig[0].azure.azure_nodepool_client_id = \"${AZURE_NODEPOOL_CLIENT_ID}\"" \
        ".providerConfig[0].azure.jfrog_oidc_provider_name = \"${JFROG_OIDC_PROVIDER_NAME}\"" \

    run_helm_test \
        "${release_name}" \
        "${namespace}" \
        "${values_file}" \
        "${TEST_IMAGE}" \
        "jfrog-test" \
        "${node_label_value}" \
        "false" \

    log_info "TEST PASSED: Azure OIDC"

    cleanup_azure
    trap - EXIT
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    log_step "Starting Azure E2E tests (run: ${RUN_ID})"

    connect_cluster_azure "${AKS_CLUSTER_NAME}" "${AKS_RESOURCE_GROUP}"

    test_azure_oidc

    log_step "All Azure E2E tests PASSED"
}

main "$@"
