Content-Type: multipart/mixed; boundary="//"
MIME-Version: 1.0

--//
Content-Transfer-Encoding: 7bit
Content-Type: text/x-shellscript
Mime-Version: 1.0


echo '{
  "name": "jfrog-credential-provider",
  "matchImages": [
    "*.jfrog.io"
  ],
  "defaultCacheDuration": "5h",
  "apiVersion": "credentialprovider.kubelet.k8s.io/v1",
  "env": [
    {
      "name": "artifactory_url",
      "value": "__TARGET_ARTIFACTORY_URL__"
    },
    {
      "name": "aws_auth_method",
      "value": "assume_role"
    },
    {
      "name": "aws_role_name",
      "value": "__AWS_EC2_ROLE_NAME__"
    }
  ]
} 

' > /etc/eks/image-credential-provider/jfrog-provider.json

export JFROG_CREDENTIAL_PROVIDER_BINARY_URL="__JFROG_CREDENTIAL_PROVIDER_BINARY_URL__"
export ARTIFACTORY_URL="__TARGET_ARTIFACTORY_URL__"
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
    ${IMAGE_CREDENTIAL_PROVIDER_DIR}/jfrog-credential-provider add-provider-config

fi
--//--