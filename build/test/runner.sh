#!/bin/bash
# runner.sh - YAML-driven E2E test runner
#
# Usage:
#   runner.sh --provider aws                      # run all AWS tests
#   runner.sh --provider aws --parallel            # run all AWS tests in parallel
#   runner.sh --provider all --parallel            # run every test in parallel
#   runner.sh build/test/cases/aws-assume-role.yaml  # run a single case file
#   runner.sh build/test/cases/aws-*.yaml            # run matching cases via glob
#   runner.sh build/test/cases/                      # run all cases in directory
#
# Local example:
#   source build/test/env && bash build/test/runner.sh --provider aws
#
# Namespace and releaseName are auto-generated from the case name + RUN_ID.
# Requires: yq, helm, kubectl, and the relevant cloud CLI (aws/az/gcloud).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/helper.sh"

RUN_ID="${GITHUB_RUN_ID:-$(date +%s)}"
SHORT_RUN_ID="${RUN_ID: -8}"

HELM_REPO_NAME="jfrog"
HELM_REPO_URL="https://charts.jfrog.io/"
HELM_CHART_NAME="jfrog-credential-provider"
HELM_REPO_ADDED=false

# ---------------------------------------------------------------------------
# Expand env vars inside a string (safe subset via envsubst)
# ---------------------------------------------------------------------------
expand_env() {
    echo "$1" | envsubst
}

# ---------------------------------------------------------------------------
# Read a YAML field, expand env vars, return empty string on null
# ---------------------------------------------------------------------------
yaml_field() {
    local file="$1"
    local path="$2"
    local raw
    raw=$(yq -r "${path} // \"\"" "${file}")
    expand_env "${raw}"
}

# ---------------------------------------------------------------------------
# Resolve chart reference for a step (local dir or released repo chart)
# ---------------------------------------------------------------------------
resolve_chart_ref() {
    local case_file="$1"
    local step_index="$2"

    local chart_source
    chart_source=$(yaml_field "${case_file}" ".steps[${step_index}].helm.chartSource")

    if [[ "${chart_source}" == "released" ]]; then
        if [[ "${HELM_REPO_ADDED}" != true ]]; then
            log_info "Adding Helm repo ${HELM_REPO_NAME} -> ${HELM_REPO_URL}" >&2
            helm repo add "${HELM_REPO_NAME}" "${HELM_REPO_URL}" --force-update >&2
            helm repo update "${HELM_REPO_NAME}" >&2
            HELM_REPO_ADDED=true
        fi

        local chart_version
        chart_version=$(yaml_field "${case_file}" ".steps[${step_index}].helm.chartVersion")

        if [[ -n "${chart_version}" ]]; then
            echo "--version ${chart_version} ${HELM_REPO_NAME}/${HELM_CHART_NAME}"
        else
            echo "${HELM_REPO_NAME}/${HELM_CHART_NAME}"
        fi
    else
        echo "${REPO_ROOT}/helm"
    fi
}

# ---------------------------------------------------------------------------
# Provider dispatch: connect
# ---------------------------------------------------------------------------
connect_cluster() {
    local provider="$1"
    case "${provider}" in
        aws)
            connect_cluster_aws "${EKS_CLUSTER_NAME}" "${AWS_REGION}"
            ;;
        azure)
            connect_cluster_azure "${AKS_CLUSTER_NAME}" "${AKS_RESOURCE_GROUP}"
            ;;
        gcp)
            connect_cluster_gcp "${GKE_CLUSTER_NAME}" "${GCP_PROJECT}" "${GCP_ZONE}"
            ;;
        *)
            log_error "Unknown provider: ${provider}"
            return 1
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Provider dispatch: create node group
# ---------------------------------------------------------------------------
create_node_group() {
    local provider="$1"
    local case_file="$2"
    local ng_name="$3"
    local test_name="$4"

    local extra_labels
    extra_labels=$(yaml_field "${case_file}" '.nodeGroup.labels')
    # Auto-prepend jfrog-test=<name> and credentialsProviderEnabled=true
    local labels="jfrog-test=${test_name},credentialsProviderEnabled=true"
    if [[ -n "${extra_labels}" ]]; then
        labels="${labels},${extra_labels}"
    fi

    case "${provider}" in
        aws)
            local instance_type ami_type
            instance_type=$(yaml_field "${case_file}" '.nodeGroup.instanceType')
            ami_type=$(yaml_field "${case_file}" '.nodeGroup.amiType')

            create_node_group_aws \
                "${EKS_CLUSTER_NAME}" \
                "${ng_name}" \
                "${labels}" \
                "${instance_type}" \
                "${AWS_SUBNET_IDS}" \
                "${ami_type}" \
                "${AWS_NODE_ROLE_ARN}"
            ;;
        azure)
            local vm_size node_count identity_client_id
            vm_size=$(yaml_field "${case_file}" '.nodeGroup.vmSize')
            vm_size="${vm_size:-Standard_D2pds_v5}"
            node_count=$(yaml_field "${case_file}" '.nodeGroup.nodeCount')
            node_count="${node_count:-1}"
            identity_client_id=$(yaml_field "${case_file}" '.nodeGroup.identityClientId')
            identity_client_id=$(expand_env "${identity_client_id}")

            create_node_group_azure \
                "${AKS_CLUSTER_NAME}" \
                "${AKS_RESOURCE_GROUP}" \
                "${ng_name}" \
                "${labels}" \
                "${vm_size}" \
                "${node_count}" \
                "${identity_client_id}"
            ;;
        gcp)
            local machine_type num_nodes service_account_email
            machine_type=$(yaml_field "${case_file}" '.nodeGroup.machineType')
            machine_type="${machine_type:-e2-medium}"
            num_nodes=$(yaml_field "${case_file}" '.nodeGroup.numNodes')
            num_nodes="${num_nodes:-1}"
            service_account_email=$(yaml_field "${case_file}" '.nodeGroup.serviceAccountEmail')
            service_account_email=$(expand_env "${service_account_email}")

            create_node_group_gcp \
                "${GKE_CLUSTER_NAME}" \
                "${GCP_PROJECT}" \
                "${GCP_ZONE}" \
                "${ng_name}" \
                "${labels}" \
                "${machine_type}" \
                "${num_nodes}" \
                "${service_account_email}"
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Provider dispatch: delete node group
# ---------------------------------------------------------------------------
delete_node_group() {
    local provider="$1"
    local ng_name="$2"

    case "${provider}" in
        aws)
            delete_node_group_aws "${EKS_CLUSTER_NAME}" "${ng_name}"
            ;;
        azure)
            delete_node_group_azure "${AKS_CLUSTER_NAME}" "${AKS_RESOURCE_GROUP}" "${ng_name}"
            ;;
        gcp)
            delete_node_group_gcp "${GKE_CLUSTER_NAME}" "${GCP_PROJECT}" "${GCP_ZONE}" "${ng_name}"
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Run a single test case file
# ---------------------------------------------------------------------------
run_case() {
    local case_file="$1"

    log_step "Running test case: ${case_file}"

    local name provider
    name=$(yaml_field "${case_file}" '.name')
    provider=$(yaml_field "${case_file}" '.provider')

    # Auto-generate namespace, release name, and node group name from case name
    local namespace="${name}-${SHORT_RUN_ID}-$((RANDOM % 10))"
    local release_name="${name}"
    local ng_name="${name}-${SHORT_RUN_ID}-$((RANDOM % 10))"

    # Azure node pool names must be <= 12 chars, alphanumeric only
    if [[ "${provider}" == "azure" ]]; then
        local name_stripped="${name//[^a-z0-9]/}"
        ng_name="${name_stripped:0:4}${SHORT_RUN_ID:0:4}$((RANDOM % 10))"
    fi

    log_info "namespace=${namespace}  release=${release_name}  nodeGroup=${ng_name}"

    local step_count
    step_count=$(yq '.steps | length' "${case_file}")
    log_info "Test '${name}' (${provider}) has ${step_count} step(s)"

    # --- cleanup handler ---
    cleanup_case() {
        log_step "Cleanup: ${name}"
        cleanup_helm_test "${release_name}" "${namespace}" || true
        delete_node_group "${provider}" "${ng_name}" || true
    }
    trap cleanup_case EXIT

    # --- infra setup ---
    connect_cluster "${provider}"
    create_node_group "${provider}" "${case_file}" "${ng_name}" "${name}"

    # --- step loop ---
    for i in $(seq 0 $((step_count - 1))); do
        local action base_values values_file chart_ref
        action=$(yq -r ".steps[${i}].action" "${case_file}")
        base_values=$(yaml_field "${case_file}" ".steps[${i}].helm.baseValues")
        values_file="/tmp/values-${name}-step${i}.yaml"
        chart_ref=$(resolve_chart_ref "${case_file}" "${i}")

        log_step "Step $((i + 1))/${step_count}: ${action} (chart: ${chart_ref})"

        # Build overrides array
        local override_count
        override_count=$(yq ".steps[${i}].helm.overrides | length" "${case_file}")
        local -a overrides=()
        for j in $(seq 0 $((override_count - 1))); do
            local raw_expr
            raw_expr=$(yq -r ".steps[${i}].helm.overrides[${j}]" "${case_file}")
            overrides+=("$(expand_env "${raw_expr}")")
        done

        # Pin the chart's DaemonSet to the test node group via nodeSelector
        overrides+=(".nodeSelector.\"jfrog-test\" = \"${name}\"")

        generate_values "${REPO_ROOT}/${base_values}" "${values_file}" "${overrides[@]}"

        case "${action}" in
            install)
                helm_install "${release_name}" "${namespace}" "${values_file}" "${chart_ref}"
                ;;
            upgrade)
                helm_upgrade "${release_name}" "${namespace}" "${values_file}" "${chart_ref}"
                ;;
            *)
                log_error "Unknown step action: ${action}"
                return 1
                ;;
        esac

        # Verify: deploy test pod and wait
        local test_image projected_token pod_name
        test_image=$(yaml_field "${case_file}" ".steps[${i}].verify.image")
        projected_token=$(yaml_field "${case_file}" ".steps[${i}].verify.projectedToken")
        projected_token="${projected_token:-false}"
        pod_name="test-${release_name}-s${i}"

        # Clean up previous step's test pod if any
        if [[ "${i}" -gt 0 ]]; then
            local prev_pod="test-${release_name}-s$((i - 1))"
            cleanup_test_pod "${prev_pod}" "${namespace}" || true
        fi

        deploy_test_pod "${pod_name}" "${namespace}" "${test_image}" "jfrog-test" "${name}" "${projected_token}"
        wait_for_pod "${pod_name}" "${namespace}"

        log_info "Step $((i + 1)) PASSED: ${action}"
    done

    log_info "TEST PASSED: ${name}"

    cleanup_case
    trap - EXIT
}

# ---------------------------------------------------------------------------
# Main: resolve arguments into case files and run each
# ---------------------------------------------------------------------------
main() {
    local parallel=false
    local provider=""

    if [[ $# -eq 0 ]]; then
        echo "Usage: $0 [--parallel] [--provider aws|azure|gcp|all] [case-file.yaml ...]" >&2
        exit 1
    fi

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --parallel) parallel=true; shift ;;
            --provider) provider="$2"; shift 2 ;;
            *) break ;;
        esac
    done

    local -a case_files=()

    if [[ -n "${provider}" ]]; then
        local cases_dir="${SCRIPT_DIR}/cases"
        if [[ "${provider}" == "all" ]]; then
            for f in "${cases_dir}"/*.yaml; do
                [[ -f "${f}" ]] && case_files+=("${f}")
            done
        else
            for f in "${cases_dir}"/${provider}-*.yaml; do
                [[ -f "${f}" ]] && case_files+=("${f}")
            done
        fi
    fi

    for arg in "$@"; do
        if [[ -d "${arg}" ]]; then
            for f in "${arg}"/*.yaml; do
                [[ -f "${f}" ]] && case_files+=("${f}")
            done
        elif [[ -f "${arg}" ]]; then
            case_files+=("${arg}")
        else
            log_error "No such file or directory: ${arg}"
            exit 1
        fi
    done

    if [[ ${#case_files[@]} -eq 0 ]]; then
        log_error "No test case files found"
        exit 1
    fi

    if [[ "${parallel}" == true ]] && [[ ${#case_files[@]} -gt 1 ]]; then
        log_step "Running ${#case_files[@]} test case(s) in PARALLEL"
        local -a pids=()
        local -a names=()
        for cf in "${case_files[@]}"; do
            ( run_case "${cf}" ) &
            pids+=($!)
            names+=("${cf}")
        done

        local failed=0
        for i in "${!pids[@]}"; do
            if ! wait "${pids[$i]}"; then
                log_error "FAILED: ${names[$i]}"
                failed=$((failed + 1))
            else
                log_info "PASSED: ${names[$i]}"
            fi
        done

        if [[ "${failed}" -gt 0 ]]; then
            log_error "${failed}/${#case_files[@]} test case(s) FAILED"
            exit 1
        fi
    else
        log_step "Running ${#case_files[@]} test case(s) sequentially"
        for cf in "${case_files[@]}"; do
            run_case "${cf}"
        done
    fi

    log_step "All test cases PASSED"
}

main "$@"
