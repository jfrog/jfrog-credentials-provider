#!/bin/bash

# This script will download the jfrog-credential-provider and setup the needed configuration before the kubelet starts.
# The code here runs on the EKS node as part of the bootstrap process, before starting the kubelet.

######################## TEMP ######################
#echo "#### Adding temp public key to allow SSH access from bastion"
#echo '<the ssh public key to inject to the nodes>' >> ~/.ssh/authorized_keys
#
#echo "#### ~/.ssh/authorized_keys"
#cat ~/.ssh/authorized_keys
#echo "############################################################"
######################## TEMP ######################


export IMAGE_CREDENTIAL_PROVIDER_DIR=/etc/eks/image-credential-provider

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
