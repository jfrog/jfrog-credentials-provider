# Debugging Guide

## Quick Debugging Steps

### 1. Access the Node

First, identify the node where your plugin should be running:

```bash
kubectl get nodes --show-labels | grep "YOUR_NODE_LABEL"
```

Create a privileged debug pod to access the node's filesystem:

```bash
kubectl debug node/NODE_NAME -it --image=ubuntu -- bash
```

Once inside the pod, access the host filesystem:

```bash
chroot /host
```

### 2. Check Plugin Installation

**Plugin Location**: 
 AWS: `/etc/eks/image-credential-provider/jfrog-credential-provider`
 Azure: `/var/lib/kubelet/credential-provider/jfrog-credential-provider`
 GCP: `/home/kubernetes/bin/jfrog-credential-provider`

**Configuration**: 
 AWS: `/etc/eks/image-credential-provider/config.json`
 Azure: `/var/lib/kubelet/credential-provider-config.yaml`
 GCP: `/etc/srv/kubernetes/cri_auth_config.yaml`

**Logs**: `/var/log/jfrog-credential-provider.log`

### 3. Test the Plugin Manually

Create a test request file:

```bash
cat > request.json << EOF
{
    "apiVersion": "credentialprovider.kubelet.k8s.io/v1",
    "kind": "CredentialProviderRequest",
    "image": "YOUR-ARTIFACTORY-URL/image:tag"
}
EOF
```

Export required environment variables (check config file for your specific cloud provider for your specific variables):

Example for AWS:

```bash
# For assume_role method:
export aws_role_name=YOUR_ROLE_NAME
export artifactory_url=YOUR_ARTIFACTORY_URL
export aws_auth_method=assume_role

# For OIDC method:
export aws_auth_method=cognito_oidc
export jfrog_oidc_provider_name=YOUR_PROVIDER_NAME
# ... other OIDC variables
```

Example for Azure:

```bash
export artifactory_url=YOUR_ARTIFACTORY_URL
export azure_app_client_id=YOUR_AZURE_APP_CLIENT_ID
export azure_tenant_id=YOUR_AZURE_TENANT_ID
export azure_nodepool_client_id=YOUR_AZURE_NODEPOOL_CLIENT_ID
export azure_app_audience=api://AzureADTokenExchange
export jfrog_oidc_provider_name=YOUR_PROVIDER_NAME
```

Example for GCP:

```bash
export artifactory_url=YOUR_ARTIFACTORY_URL
export google_service_account_email=YOUR_SERVICE_ACCOUNT_EMAIL
export jfrog_oidc_audience=YOUR_GCP_PROJECT_ID
export jfrog_oidc_provider_name=YOUR_PROVIDER_NAME
```

Run the plugin manually:

```bash
./jfrog-credentials-provider < request.json
```

This should output a `CredentialProviderResponse` with an auth token if successful.

### 4. Check Installation Issues

#### AWS

If the plugin wasn't deployed at all, check the user data script execution:

**User Data Script**: 
```bash
cat /var/lib/cloud/instance/user-data.txt
```

**Installation Logs**:
```bash
# Check for errors in cloud-init logs
grep -i error /var/log/cloud-init-output.log
grep -i error /var/log/cloud-init.log

# Or manually review the logs
less /var/log/cloud-init-output.log
```

## Common Issues

- **MIME syntax errors** in Launch Template user data
- **Missing environment variables** in configuration
- **AWS permissions** for the node's IAM role
- **JFrog Artifactory** user/role mapping not configured
- **Network connectivity** issues to cloud services or Artifactory

## Log Locations

- **Plugin logs**: `/var/log/jfrog-credential-provider.log`
- **Kubelet logs**: `journalctl -u kubelet`