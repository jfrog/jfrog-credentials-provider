#!/bin/bash
# helper.sh - Shared library for JFrog Credential Provider E2E tests
# Sourced by aws.sh, azure.sh, gcp.sh

set -euo pipefail

# ---------------------------------------------------------------------------
# Globals
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LOG_FILE="/tmp/jfrog-test-${GITHUB_RUN_ID:-local}.log"
POD_WAIT_TIMEOUT="${POD_WAIT_TIMEOUT:-300}"
DAEMONSET_WAIT_TIMEOUT="${DAEMONSET_WAIT_TIMEOUT:-300}"
NODE_GROUP_WAIT_TIMEOUT="${NODE_GROUP_WAIT_TIMEOUT:-600}"

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
_log() {
    local level="$1"; shift
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] [${level}] $*"
    echo "${msg}" | tee -a "${LOG_FILE}"
}

log_info()  { _log "INFO"  "$@"; }
log_warn()  { _log "WARN"  "$@"; }
log_error() { _log "ERROR" "$@"; }
log_step()  { _log "STEP"  "===== $* ====="; }

# ---------------------------------------------------------------------------
# Cluster connection
# ---------------------------------------------------------------------------
connect_cluster_aws() {
    local cluster_name="$1"
    local region="$2"

    log_step "Connecting to EKS cluster ${cluster_name} in ${region}"
    if ! aws eks update-kubeconfig --name "${cluster_name}" --region "${region}"; then
        log_error "Failed to connect to EKS cluster ${cluster_name}"
        return 1
    fi
    log_info "Connected to EKS cluster ${cluster_name}"
}

connect_cluster_azure() {
    local cluster_name="$1"
    local resource_group="$2"

    log_step "Connecting to AKS cluster ${cluster_name} in resource group ${resource_group}"
    if ! az aks get-credentials --name "${cluster_name}" --resource-group "${resource_group}" --overwrite-existing; then
        log_error "Failed to connect to AKS cluster ${cluster_name}"
        return 1
    fi
    log_info "Connected to AKS cluster ${cluster_name}"
}

connect_cluster_gcp() {
    local cluster_name="$1"
    local project="$2"
    local zone="$3"

    log_step "Connecting to GKE cluster ${cluster_name} in project ${project}, zone ${zone}"
    if ! gcloud container clusters get-credentials "${cluster_name}" --project "${project}" --zone "${zone}"; then
        log_error "Failed to connect to GKE cluster ${cluster_name}"
        return 1
    fi
    log_info "Connected to GKE cluster ${cluster_name}"
}

# ---------------------------------------------------------------------------
# Node group creation
# ---------------------------------------------------------------------------

# create_node_group_aws CLUSTER_NAME NG_NAME LABELS INSTANCE_TYPE SUBNET_IDS NODE_ROLE_ARN [MIN] [MAX] [DESIRED]
# LABELS is a comma-separated key=value string, e.g. "jfrog-test=aws-assume-role,env=ci"
create_node_group_aws() {
    local cluster_name="$1"
    local ng_name="$2"
    local labels="$3"
    local instance_type="$4"
    local subnet_ids="$5"
    local ami_type="$6"
    local node_role_arn="$7"
    local min_size="${8:-1}"
    local max_size="${9:-1}"
    local desired_size="${10:-1}"

    log_step "Creating AWS node group ${ng_name} on cluster ${cluster_name}"
    log_info "Labels: ${labels} | Instance: ${instance_type} | Size: ${min_size}/${max_size}/${desired_size}"

    if ! aws eks create-nodegroup \
        --cluster-name "${cluster_name}" \
        --nodegroup-name "${ng_name}" \
        --node-role "${node_role_arn}" \
        --labels "${labels}" \
        --instance-types "${instance_type}" \
        --subnets ${subnet_ids} \
        --ami-type "${ami_type}" \
        --scaling-config "minSize=${min_size},maxSize=${max_size},desiredSize=${desired_size}"; then
        log_error "Failed to create AWS node group ${ng_name}"
        return 1
    fi

    log_info "Waiting for node group ${ng_name} to become ACTIVE (timeout: ${NODE_GROUP_WAIT_TIMEOUT}s)..."
    if ! aws eks wait nodegroup-active \
        --cluster-name "${cluster_name}" \
        --nodegroup-name "${ng_name}"; then
        log_error "Node group ${ng_name} did not reach ACTIVE state"
        return 1
    fi
    log_info "Node group ${ng_name} is ACTIVE"
}

# create_node_group_azure CLUSTER_NAME RESOURCE_GROUP NG_NAME LABELS VM_SIZE [NODE_COUNT] [IDENTITY_CLIENT_ID]
# LABELS is a comma-separated key=value string, e.g. "jfrog-test=azure-oidc"
create_node_group_azure() {
    local cluster_name="$1"
    local resource_group="$2"
    local ng_name="$3"
    local labels="$4"
    local vm_size="$5"
    local node_count="${6:-1}"
    local identity_client_id="${7:-}"

    log_step "Creating Azure node pool ${ng_name} on cluster ${cluster_name}"
    log_info "Labels: ${labels} | VM size: ${vm_size} | Count: ${node_count}"

    # Convert comma-separated "k=v,k2=v2" to space-separated "k=v k2=v2" for az CLI
    local label_args="${labels//,/ }"

    if ! az aks nodepool add \
        --cluster-name "${cluster_name}" \
        --resource-group "${resource_group}" \
        --name "${ng_name}" \
        --labels ${label_args} \
        --node-count "${node_count}" \
        --node-vm-size "${vm_size}"; then
        log_error "Failed to create Azure node pool ${ng_name}"
        return 1
    fi
    log_info "Azure node pool ${ng_name} is ready"

    if [[ -n "${identity_client_id}" ]]; then
        log_info "Assigning managed identity ${identity_client_id} to node pool VMSS"

        local mc_rg
        mc_rg=$(az aks show \
            --name "${cluster_name}" \
            --resource-group "${resource_group}" \
            --query "nodeResourceGroup" -o tsv)

        local vmss_name
        vmss_name=$(az vmss list \
            -g "${mc_rg}" \
            --query "[?tags.\"aks-managed-poolName\"=='${ng_name}'].name" -o tsv)

        if [[ -z "${vmss_name}" ]]; then
            log_error "Could not find VMSS for node pool ${ng_name} in ${mc_rg}"
            return 1
        fi

        local identity_id
        identity_id=$(az identity list \
            --query "[?clientId=='${identity_client_id}'].id" -o tsv)

        if [[ -z "${identity_id}" ]]; then
            log_error "Could not find managed identity with clientId ${identity_client_id}"
            return 1
        fi

        if ! az vmss identity assign \
            -g "${mc_rg}" \
            -n "${vmss_name}" \
            --identities "${identity_id}"; then
            log_error "Failed to assign identity to Azure node pool ${ng_name}"
            return 1
        fi
        log_info "Identity assigned to Azure node pool ${ng_name}"
    fi
}

# create_node_group_gcp CLUSTER_NAME PROJECT ZONE NG_NAME LABELS MACHINE_TYPE [NUM_NODES]
# LABELS is a comma-separated key=value string, e.g. "jfrog-test=gcp-oidc"
create_node_group_gcp() {
    local cluster_name="$1"
    local project="$2"
    local zone="$3"
    local ng_name="$4"
    local labels="$5"
    local machine_type="$6"
    local num_nodes="${7:-1}"
    local service_account_email="$8"

    log_step "Creating GCP node pool ${ng_name} on cluster ${cluster_name}"
    log_info "Labels: ${labels} | Machine type: ${machine_type} | Nodes: ${num_nodes}"

    if ! gcloud container node-pools create "${ng_name}" \
        --cluster "${cluster_name}" \
        --project "${project}" \
        --zone "${zone}" \
        --machine-type "${machine_type}" \
        --num-nodes "${num_nodes}" \
        --node-labels "${labels}" \
        --pod-ipv4-range "pod-ranges-extra" \
        --service-account "${service_account_email}"; then
        log_error "Failed to create GCP node pool ${ng_name}"
        return 1
    fi
    log_info "GCP node pool ${ng_name} is ready"
}

# ---------------------------------------------------------------------------
# Node group deletion
# ---------------------------------------------------------------------------
delete_node_group_aws() {
    local cluster_name="$1"
    local ng_name="$2"

    log_step "Deleting AWS node group ${ng_name} from cluster ${cluster_name}"
    if ! aws eks delete-nodegroup --cluster-name "${cluster_name}" --nodegroup-name "${ng_name}"; then
        log_warn "Failed to delete AWS node group ${ng_name} (may not exist)"
        return 0
    fi
    log_info "Waiting for node group ${ng_name} to be deleted..."
    aws eks wait nodegroup-deleted --cluster-name "${cluster_name}" --nodegroup-name "${ng_name}" || true
    log_info "AWS node group ${ng_name} deleted"
}

delete_node_group_azure() {
    local cluster_name="$1"
    local resource_group="$2"
    local ng_name="$3"

    log_step "Deleting Azure node pool ${ng_name} from cluster ${cluster_name}"
    if ! az aks nodepool delete \
        --cluster-name "${cluster_name}" \
        --resource-group "${resource_group}" \
        --name "${ng_name}" \
        --no-wait; then
        log_warn "Failed to delete Azure node pool ${ng_name} (may not exist)"
        return 0
    fi
    log_info "Azure node pool ${ng_name} deleted"
}

delete_node_group_gcp() {
    local cluster_name="$1"
    local project="$2"
    local zone="$3"
    local ng_name="$4"

    log_step "Deleting GCP node pool ${ng_name} from cluster ${cluster_name}"
    if ! gcloud container node-pools delete "${ng_name}" \
        --cluster "${cluster_name}" \
        --project "${project}" \
        --zone "${zone}" \
        --quiet; then
        log_warn "Failed to delete GCP node pool ${ng_name} (may not exist)"
        return 0
    fi
    log_info "GCP node pool ${ng_name} deleted"
}

# ---------------------------------------------------------------------------
# Values generation
# ---------------------------------------------------------------------------

# generate_values TEMPLATE_FILE OUTPUT_FILE [YQ_EXPR...]
# Copies the template then applies each yq expression in order.
# Example:
#   generate_values examples/aws-values.yaml /tmp/out.yaml \
#       '.providerConfig[0].artifactoryUrl = "my.jfrog.io"' \
#       '.nodeSelector."jfrog-test" = "aws-assume-role"'
generate_values() {
    local template_file="$1"
    local output_file="$2"
    shift 2

    log_info "Generating values file from ${template_file} -> ${output_file}"
    cp "${template_file}" "${output_file}"

    for expr in "$@"; do
        log_info "  yq: ${expr}"
        yq -i "${expr}" "${output_file}"
    done

    log_info "Generated values file: ${output_file}"
}

# ---------------------------------------------------------------------------
# Helm lifecycle
# ---------------------------------------------------------------------------

# helm_install RELEASE_NAME NAMESPACE VALUES_FILE CHART_VERSION
helm_install() {
    local release_name="$1"
    local namespace="$2"
    local values_file="$3"

    log_step "Helm installing ${release_name} in namespace ${namespace})"


    if ! helm install "${release_name}" "${REPO_ROOT}/helm" \
        --namespace "${namespace}" \
        --create-namespace \
        -f "${values_file}" \
        --wait \
        --timeout "${DAEMONSET_WAIT_TIMEOUT}s"; then
        log_error "Helm install failed for ${release_name}"
        log_info "Dumping pod status in namespace ${namespace}:"
        kubectl get pods -n "${namespace}" -o wide 2>&1 | tee -a "${LOG_FILE}" || true
        log_info "Dumping events in namespace ${namespace}:"
        kubectl get events -n "${namespace}" --sort-by='.lastTimestamp' 2>&1 | tee -a "${LOG_FILE}" || true
        return 1
    fi

    log_info "Helm install succeeded for ${release_name}"
    log_info "Pods in namespace ${namespace}:"
    kubectl get pods -n "${namespace}" -o wide 2>&1 | tee -a "${LOG_FILE}"
}

# helm_uninstall RELEASE_NAME NAMESPACE
helm_uninstall() {
    local release_name="$1"
    local namespace="$2"

    log_step "Helm uninstalling ${release_name} from namespace ${namespace}"
    if ! helm uninstall "${release_name}" --namespace "${namespace}"; then
        log_warn "Helm uninstall failed for ${release_name} (may not exist)"
        return 0
    fi
    log_info "Helm uninstall succeeded for ${release_name}"
}

# ---------------------------------------------------------------------------
# Pod verification
# ---------------------------------------------------------------------------

# deploy_test_pod POD_NAME NAMESPACE IMAGE NODE_SELECTOR_KEY NODE_SELECTOR_VALUE
# Creates a minimal pod that pulls from the target Artifactory to validate credentials.
deploy_test_pod() {
    local pod_name="$1"
    local namespace="$2"
    local image="$3"
    local node_selector_key="$4"
    local node_selector_value="$5"
    local projected_token_enabled="$6"

    if [[ "${projected_token_enabled}" == "true" ]]; then
        localservice_account_name="projected-sa"
        kubectl create serviceaccount ${service_account_name} -n ${namespace}
        kubectl annotate serviceaccount ${service_account_name} -n ${namespace} "eks.amazonaws.com/role-arn=${node_role_arn}"
    else
        service_account_name="default"
    fi

    log_step "Deploying test pod ${pod_name} in ${namespace} (image: ${image})"

    kubectl apply -n "${namespace}" -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${pod_name}
  namespace: ${namespace}
  labels:
    app: jfrog-credential-provider-test
spec:
  serviceAccountName: ${service_account_name}
  nodeSelector:
    ${node_selector_key}: "${node_selector_value}"
  containers:
    - name: test
      image: ${image}
      command: ["sh", "-c", "echo 'Image pull succeeded - credential provider works' && sleep 3600"]
  restartPolicy: Never
  terminationGracePeriodSeconds: 5
EOF

    log_info "Test pod ${pod_name} manifest applied"
}

# wait_for_pod POD_NAME NAMESPACE [TIMEOUT]
# Polls until pod is Running or timeout is reached.
wait_for_pod() {
    local pod_name="$1"
    local namespace="$2"
    local timeout="${3:-${POD_WAIT_TIMEOUT}}"

    log_info "Waiting for pod ${pod_name} to be Running (timeout: ${timeout}s)..."

    local elapsed=0
    local interval=10
    while [ "${elapsed}" -lt "${timeout}" ]; do
        local phase
        phase=$(kubectl get pod "${pod_name}" -n "${namespace}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

        case "${phase}" in
            Running)
                log_info "Pod ${pod_name} is Running"
                return 0
                ;;
            Succeeded)
                log_info "Pod ${pod_name} Succeeded (completed)"
                return 0
                ;;
            Failed)
                log_error "Pod ${pod_name} has Failed"
                kubectl describe pod "${pod_name}" -n "${namespace}" 2>&1 | tee -a "${LOG_FILE}" || true
                return 1
                ;;
            *)
                log_info "Pod ${pod_name} phase: ${phase} (${elapsed}s/${timeout}s)"
                ;;
        esac

        sleep "${interval}"
        elapsed=$((elapsed + interval))
    done

    log_error "Timed out waiting for pod ${pod_name} to reach Running state"
    kubectl describe pod "${pod_name}" -n "${namespace}" 2>&1 | tee -a "${LOG_FILE}" || true
    kubectl get events -n "${namespace}" --field-selector "involvedObject.name=${pod_name}" --sort-by='.lastTimestamp' 2>&1 | tee -a "${LOG_FILE}" || true
    return 1
}

# cleanup_test_pod POD_NAME NAMESPACE
cleanup_test_pod() {
    local pod_name="$1"
    local namespace="$2"

    log_info "Cleaning up test pod ${pod_name} from namespace ${namespace}"
    kubectl delete pod "${pod_name}" -n "${namespace}" --ignore-not-found --grace-period=5 || true
    log_info "Test pod ${pod_name} cleaned up"
}

# ---------------------------------------------------------------------------
# Full test lifecycle helper
# ---------------------------------------------------------------------------

# run_helm_test RELEASE_NAME NAMESPACE VALUES_FILE CHART_VERSION TEST_IMAGE NODE_SELECTOR_KEY NODE_SELECTOR_VALUE
# Runs a complete helm install -> deploy test pod -> verify -> cleanup cycle.
run_helm_test() {
    local release_name="$1"
    local namespace="$2"
    local values_file="$3"
    local test_image="$4"
    local node_selector_key="$5"
    local node_selector_value="$6"
    local projected_token_enabled="$7"
    local pod_name="test-${release_name}"

    log_step "Starting helm test for ${release_name}"

    helm_install "${release_name}" "${namespace}" "${values_file}"

    deploy_test_pod "${pod_name}" "${namespace}" "${test_image}" "${node_selector_key}" "${node_selector_value}" "${projected_token_enabled}"
    wait_for_pod "${pod_name}" "${namespace}"

    log_info "Helm test PASSED for ${release_name}"
}

# cleanup_helm_test RELEASE_NAME NAMESPACE
# Cleans up a helm release and its test pod.
cleanup_helm_test() {
    local release_name="$1"
    local namespace="$2"
    local pod_name="test-${release_name}"

    log_step "Cleaning up helm test for ${release_name}"
    cleanup_test_pod "${pod_name}" "${namespace}"
    helm_uninstall "${release_name}" "${namespace}"
    kubectl delete namespace "${namespace}" --ignore-not-found || true
    log_info "Cleanup complete for ${release_name}"
}

log_info "helper.sh loaded (log file: ${LOG_FILE})"
