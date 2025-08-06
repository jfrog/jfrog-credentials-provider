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

**Plugin Location**: `/etc/eks/image-credential-provider/jfrog-credential-provider`

**Configuration**: `/etc/eks/image-credential-provider/config.json`

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

Export required environment variables (check `/etc/eks/image-credential-provider/config.json` for your specific variables):

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

Run the plugin manually:

```bash
./jfrog-credential-provider < request.json
```

This should output a `CredentialProviderResponse` with an auth token if successful.

### 4. Check Installation Issues

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

## Log Locations

- **Plugin logs**: `/var/log/jfrog-credential-provider.log`
- **Cloud-init logs**: `/var/log/cloud-init-output.log` and `/var/log/cloud-init.log`
- **Kubelet logs**: `journalctl -u kubelet`