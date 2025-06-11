#!/bin/bash
export IMAGE_CREDENTIAL_PROVIDER_DIR=/etc/eks/image-credential-provider

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
curl -s -L -f -o ${IMAGE_CREDENTIAL_PROVIDER_DIR}/jfrog-credential-provider "${JFROG_CREDENTIAL_PROVIDER_BINARY_URL}"

if [[ $? -ne 0 ]]; then
    echo "Downloading (${JFROG_CREDENTIAL_PROVIDER_BINARY_URL}) failed"
else
    echo "Successfully downloaded the jfrog-credential-provider binary from Artifactory"
    # Make the binary executable
    echo "Making the jfrog-credential-provider binary executable"
    chmod +x ${IMAGE_CREDENTIAL_PROVIDER_DIR}/jfrog-credential-provider

    # Update the kubelet configuration to use the jfrog-credential-provider
    echo "Updating the kubelet configuration to use the jfrog-credential-provider"
    jq '.providers += [input]' ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config.json ${IMAGE_CREDENTIAL_PROVIDER_DIR}/jfrog-credential-config.json > ${IMAGE_CREDENTIAL_PROVIDER_DIR}/combined-config.json

    # Replace the kubelet configuration with the updated configuration
    if [[ $? -ne 0 ]]; then
        echo "Failed to build the combined configuration, will keep using the original config.json"
    else
        echo "Overriding the default configuration with the new configuration. The original config.json is backed up as config_back.json"
        cp -f ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config.json ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config_back.json
        cp -f ${IMAGE_CREDENTIAL_PROVIDER_DIR}/combined-config.json ${IMAGE_CREDENTIAL_PROVIDER_DIR}/config.json
    fi
fi