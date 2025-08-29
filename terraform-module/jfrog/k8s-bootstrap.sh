#!/bin/bash

# This script will download the jfrog-credential-provider and setup the needed configuration.


log () {
    echo "[$(date +"%Y-%m-%d %H:%M:%S")] $1"
}

log "Startup of the JFrog Credential Plugin injector"

export IMAGE_CREDENTIAL_PROVIDER_DIR=/host/etc/eks/image-credential-provider
export IMAGE_CREDENTIAL_PROVIDER_CONFIG=/host/etc/eks/image-credential-provider/config.json

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
curl -L -f -o ${IMAGE_CREDENTIAL_PROVIDER_DIR}/jfrog-credential-provider "${JFROG_CREDENTIAL_PROVIDER_BINARY_URL}"

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
    chmod +x ${IMAGE_CREDENTIAL_PROVIDER_DIR}/jfrog-credential-provider

    echo "Copying the /etc/${IMAGE_CREDENTIAL_PROVIDER_FILE_NAME} configuration file to ${IMAGE_CREDENTIAL_PROVIDER_DIR}/${IMAGE_CREDENTIAL_PROVIDER_FILE_NAME}"
    cp -f /etc/${IMAGE_CREDENTIAL_PROVIDER_FILE_NAME} ${IMAGE_CREDENTIAL_PROVIDER_DIR}/${IMAGE_CREDENTIAL_PROVIDER_FILE_NAME}
    cat ${IMAGE_CREDENTIAL_PROVIDER_DIR}/${IMAGE_CREDENTIAL_PROVIDER_FILE_NAME}
    sleep 2 | # Wait a bit to ensure the file is copied before proceeding

    # Update the kubelet configuration to use the jfrog-credential-provider
    echo "Updating the kubelet configuration to use the jfrog-credential-provider"
    nsenter -t 1 -m -p -- /etc/eks/image-credential-provider/jfrog-credential-provider add-provider-config
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