#!/bin/bash

# This script will download the jfrog-credential-provider and setup the needed configuration.


log () {
    echo "[$(date +"%Y-%m-%d %H:%M:%S")] $1"
}

log "Startup of the JFrog Credential Plugin injector"

# KUBELET_MOUNT_PATH comes from daemonset
# Add trailing slash if not present
if [[ ${KUBELET_MOUNT_PATH} != */ ]]; then
    KUBELET_MOUNT_PATH="${KUBELET_MOUNT_PATH}/"
fi

JFROG_CONFIG_FILE="jfrog-provider"
export JFROG_CREDENTIAL_PROVIDER_BINARY_DIR=__JFROG_CREDENTIAL_PROVIDER_BINARY_DIR__

export KUBELET_CREDENTIAL_PROVIDER_CONFIG_PATH=__KUBELET_CREDENTIAL_PROVIDER_CONFIG_PATH__
# to copy the jfrog config to the kubelet config path
export KUBELET_CREDENTIAL_PROVIDER_CONFIG_DIR=$(dirname ${KUBELET_CREDENTIAL_PROVIDER_CONFIG_PATH})
# to figure out if it's yaml or json
export KUBELET_CREDENTIAL_PROVIDER_CONFIG_FILE_NAME=$(basename ${KUBELET_CREDENTIAL_PROVIDER_CONFIG_PATH})

log "The content of the current ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}:"
cat ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}

ARCH=$(uname -m)
ARCH_SUFFIX="amd64"
if [ "$ARCH" = "x86_64" ]; then
    ARCH_SUFFIX="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH_SUFFIX="arm64"
fi
export JFROG_CREDENTIAL_PROVIDER_BINARY_URL="${JFROG_CREDENTIAL_PROVIDER_BINARY_URL}-${ARCH_SUFFIX}"

# Pull the jfrog-credential-provider binary
echo "Downloading the jfrog-credential-provider binary (${JFROG_CREDENTIAL_PROVIDER_BINARY_URL})"
curl -L -f -o ${KUBELET_MOUNT_PATH}${JFROG_CREDENTIAL_PROVIDER_BINARY_DIR}/jfrog-credential-provider "${JFROG_CREDENTIAL_PROVIDER_BINARY_URL}"

if [[ $? -ne 0 ]]; then
    echo "Downloading (${JFROG_CREDENTIAL_PROVIDER_BINARY_URL}) failed"

    # Wait and exit to allow pod to restart (ugly yet simple solution for when the cluster DNS service is not ready yet)
    log "Sleeping for 10 seconds before exiting"
    sleep 10
    exit 1
else
    echo "Successfully downloaded the jfrog-credential-provider binary from Artifactory"
    # Make the binary executable
    echo "Making the jfrog-credential-provider binary executable"
    chmod +x ${KUBELET_MOUNT_PATH}${JFROG_CREDENTIAL_PROVIDER_BINARY_DIR}/jfrog-credential-provider ]]

    # if extension is yaml, set --yaml flag
    if [[ ${KUBELET_CREDENTIAL_PROVIDER_CONFIG_FILE_NAME} == *.yaml ]]; then
        echo "Copying the  /etc/${JFROG_CONFIG_FILE}.yaml configuration file to ${KUBELET_CREDENTIAL_PROVIDER_CONFIG_DIR}/${JFROG_CONFIG_FILE}.yaml"
        cp -f "/etc/${JFROG_CONFIG_FILE}.yaml" "${KUBELET_CREDENTIAL_PROVIDER_CONFIG_DIR}/${JFROG_CONFIG_FILE}.yaml"
        sleep 2 # Wait a bit to ensure the file is copied before proceeding
        nsenter -t 1 -m -p -- ${JFROG_CREDENTIAL_PROVIDER_BINARY_DIR}/jfrog-credential-provider add-provider-config --yaml --provider-home "${KUBELET_CREDENTIAL_PROVIDER_CONFIG_DIR}" --provider-config "${KUBELET_CREDENTIAL_PROVIDER_CONFIG_FILE_NAME}"
    else
        echo "Copying the  /etc/${JFROG_CONFIG_FILE}.json configuration file to ${KUBELET_MOUNT_PATH}${KUBELET_CREDENTIAL_PROVIDER_CONFIG_DIR}/${JFROG_CONFIG_FILE}.json"
        cp -f "/etc/${JFROG_CONFIG_FILE}.json" "${KUBELET_MOUNT_PATH}${KUBELET_CREDENTIAL_PROVIDER_CONFIG_DIR}/${JFROG_CONFIG_FILE}.json"
        sleep 2 # Wait a bit to ensure the file is copied before proceeding
        nsenter -t 1 -m -p -- ${JFROG_CREDENTIAL_PROVIDER_BINARY_DIR}/jfrog-credential-provider add-provider-config --provider-home "${KUBELET_CREDENTIAL_PROVIDER_CONFIG_DIR}" --provider-config "${KUBELET_CREDENTIAL_PROVIDER_CONFIG_FILE_NAME}"
    fi

    # Update the kubelet configuration to use the jfrog-credential-provider
    echo "Updating the kubelet configuration to use the jfrog-credential-provider"
    if [[ $? -ne 0 ]]; then
        echo "Updating the kubelet configuration failed"
        exit 1
    fi

    log "The final ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}:"
    cat ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}
fi

log "Done updating the kubelet config.json"


log "See kubelet service status"
nsenter -t 1 -m -p -- systemctl status kubelet

log "Restarting the kubelet service"
nsenter -t 1 -m -p -- systemctl restart kubelet

sleep 5
log "See new kubelet service status"
nsenter -t 1 -m -p -- systemctl status kubelet