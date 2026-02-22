#!/bin/bash
# aws.sh - E2E tests for AWS (assume_role + cognito_oidc)
# Expects the following env vars to be set by the caller (GitHub Actions workflow):
#   EKS_CLUSTER_NAME, AWS_REGION, AWS_SUBNET_IDS, AWS_NODE_ROLE_ARN
#   ARTIFACTORY_URL, MATCH_IMAGES, TEST_IMAGE, HELM_CHART_VERSION
#   AWS_ROLE_NAME (for assume_role)
#   AWS_COGNITO_SECRET_NAME, AWS_COGNITO_USER_POOL_NAME,
#   AWS_COGNITO_RESOURCE_SERVER_NAME, AWS_COGNITO_USER_POOL_RESOURCE_SCOPE,
#   JFROG_OIDC_PROVIDER_NAME (for cognito_oidc)
#   DOWNLOAD_URL for custom binary url

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/helper.sh"

RUN_ID="${GITHUB_RUN_ID:-$(date +%s)}"

# ---------------------------------------------------------------------------
# assume_role test
# ---------------------------------------------------------------------------
test_aws_assume_role() {
    local ng_name="jfrog-ar-${RUN_ID}"
    local release_name="jfrog-cp-assume-role"
    local namespace="jfrog-assume-role"
    local node_label_value="aws-assume-role"
    local values_file="/tmp/values-aws-assume-role.yaml"

    log_step "TEST: AWS assume_role"

    cleanup_assume_role() {
        log_step "Cleanup: AWS assume_role"
        cleanup_helm_test "${release_name}" "${namespace}" || true
        delete_node_group_aws "${EKS_CLUSTER_NAME}" "${ng_name}" || true
    }
    trap cleanup_assume_role EXIT

    create_node_group_aws \
        "${EKS_CLUSTER_NAME}" \
        "${ng_name}" \
        "jfrog-test=${node_label_value},credentialsProviderEnabled=true" \
        "t4g.medium" \
        "${AWS_SUBNET_IDS}" \
        "AL2023_ARM_64_STANDARD" \
        "${AWS_NODE_ROLE_ARN}"

    generate_values "${REPO_ROOT}/examples/aws-values.yaml" "${values_file}" \
        ".providerConfig[0].artifactoryUrl = \"${ARTIFACTORY_URL}\"" \
        ".providerConfig[0].matchImages[0] = \"${MATCH_IMAGES}\"" \
        ".providerConfig[0].aws.aws_role_name = \"${AWS_ROLE_NAME}\"" \
        ".downloadUrl = \"${DOWNLOAD_URL}\""

    run_helm_test \
        "${release_name}" \
        "${namespace}" \
        "${values_file}" \
        "${TEST_IMAGE}" \
        "jfrog-test" \
        "${node_label_value}" \
        "false" \

    log_info "TEST PASSED: AWS assume_role"

    cleanup_assume_role
    trap - EXIT
}

# ---------------------------------------------------------------------------
# cognito_oidc test
# ---------------------------------------------------------------------------
test_aws_projected_sa() {
    local ng_name="jfrog-co-${RUN_ID}"
    local release_name="jfrog-cp-projected-sa"
    local namespace="jfrog-projected-sa"
    local node_label_value="aws-projecte-token"
    local values_file="/tmp/values-aws-projected-sa.yaml"

    log_step "TEST: AWS projected_sa"

    cleanup_projected_sa() {
        log_step "Cleanup: AWS projected_sa"
        cleanup_helm_test "${release_name}" "${namespace}" || true
        delete_node_group_aws "${EKS_CLUSTER_NAME}" "${ng_name}" || true
    }
    trap cleanup_cognito EXIT

    create_node_group_aws \
        "${EKS_CLUSTER_NAME}" \
        "${ng_name}" \
        "jfrog-test=${node_label_value},credentialsProviderEnabled=true" \
        "t4g.small" \
        "${AWS_SUBNET_IDS}" \
        "${AWS_NODE_ROLE_ARN}"

    generate_values "${REPO_ROOT}/examples/aws-projected-sa-values.yaml" "${values_file}" \
        ".providerConfig[0].artifactoryUrl = \"${ARTIFACTORY_URL}\"" \
        ".providerConfig[0].matchImages[0] = \"${MATCH_IMAGES}\"" \
        ".providerConfig[0].aws.enabled = true" \
        ".providerConfig[0].aws.aws_auth_method = \"assume_role\"" \
        ".providerConfig[0].aws.aws_role_name = \"${AWS_ROLE_NAME}\"" \
        ".providerConfig[0].tokenAttributes.enabled = true" \
        ".downloadUrl = \"${DOWNLOAD_URL}\""

    run_helm_test \
        "${release_name}" \
        "${namespace}" \
        "${values_file}" \
        "${TEST_IMAGE}" \
        "jfrog-test" \
        "${node_label_value}" \
        "true" \

    log_info "TEST PASSED: AWS cognito_oidc"

    cleanup_cognito
    trap - EXIT
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    log_step "Starting AWS E2E tests (run: ${RUN_ID})"

    connect_cluster_aws "${EKS_CLUSTER_NAME}" "${AWS_REGION}"

    test_aws_assume_role
    # test_aws_cognito_oidc

    log_step "All AWS E2E tests PASSED"
}

main "$@"
