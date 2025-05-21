#!/bin/bash

# This script will download the jfrog-credential-provider and setup the needed configuration.

log () {
    echo "[$(date +"%Y-%m-%d %H:%M:%S")] $1"
}

log "Startup of the JFrog Credential Plugin injector"
cat /etc/jfrog-provider.json

export IMAGE_CREDENTIAL_PROVIDER_DIR=/host/etc/eks/image-credential-provider
export IMAGE_CREDENTIAL_PROVIDER_CONFIG=/host/etc/eks/image-credential-provider/config.json

log "The content of the current ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}:"
cat ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}

ARCH=$(uname -m)
ARCH_SUFFIX=""
if [ "$ARCH" = "x86_64" ]; then
    ARCH_SUFFIX="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH_SUFFIX="arm64"
else
    echo "Warning: Unrecognized architecture $ARCH. Defaulting to amd64 binary."
    ARCH_SUFFIX="amd64"
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

    # Update the kubelet configuration to use the jfrog-credential-provider
    echo "Updating the kubelet configuration to use the jfrog-credential-provider"
    jq '.providers += [input]' ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config.json /etc/jfrog-provider.json > ${IMAGE_CREDENTIAL_PROVIDER_DIR}/combined-config.json

    # Replace the kubelet configuration with the updated configuration
    if [[ $? -ne 0 ]]; then
        echo "Failed to build the combined configuration, will keep using the original config.json"
    else
        echo "Overriding the default configuration with the new configuration. The original config.json is backed up as config_back.json"
        cp -f ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config.json ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config_back.json
        cp -f ${IMAGE_CREDENTIAL_PROVIDER_DIR}/combined-config.json ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config.json
    fi

    log "The final ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}:"
    cat ${IMAGE_CREDENTIAL_PROVIDER_CONFIG}
fi

log "Done updating the kubelet credential-provider-config.json"


log "See kubelet service status"
nsenter -t 1 -m -p -- systemctl status kubelet

log "Restarting the kubelet service"
nsenter -t 1 -m -p -- systemctl restart kubelet

sleep 5
log "See new kubelet service status"
nsenter -t 1 -m -p -- systemctl status kubelet