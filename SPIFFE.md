# 🔐 SPIFFE JWT-SVID Setup Guide

This guide walks you through setting up the JFrog Kubelet Credential Provider using **SPIFFE JWT-SVIDs** to authenticate with JFrog Artifactory.

## 📋 Overview

Unlike the AWS / Azure / GCP guides, SPIFFE is **cloud-agnostic**. SPIFFE ([Secure Production Identity Framework For Everyone](https://spiffe.io)) is an open standard for issuing cryptographic workload identities. The credential provider fetches a short-lived **JWT-SVID** (a signed JWT that encodes a SPIFFE identity) from the node-local **SPIFFE Workload API** and exchanges it with Artifactory for a registry token — the same OIDC token-exchange used by the other providers.

**SPIFFE is the standard; the implementation is pluggable.** The Workload API socket on each node is provided by whatever SPIFFE implementation your cluster runs — for example [SPIRE](https://spiffe.io/spire/) (the CNCF reference implementation) or a commercial SPIFFE vendor. This provider targets the SPIFFE Workload API itself (via [`go-spiffe`](https://github.com/spiffe/go-spiffe)), so it works with any compliant implementation. Anywhere the implementation-specific details differ (socket path, how you enable OIDC discovery, how you fetch a test SVID) this guide calls them out as **examples**.

Because the credential provider binary is executed **on the host by the kubelet**, it can open the node's SPIFFE Workload API socket directly — no in-pod socket mount is required.

---

## ⚠️ Requirement: an OIDC-discoverable issuer (read this first)

This is the **#1 integration failure point**, so configure it before anything else.

Artifactory validates the JWT-SVID as a **"Generic OpenID Connect"** token. To do that it:

1. reads the token's **`iss` (issuer)** claim,
2. fetches `<iss>/.well-known/openid-configuration` and the JWKS it points to, then
3. verifies the token signature and claims against that key set.

The [SPIFFE JWT-SVID specification](https://github.com/spiffe/spiffe/blob/main/standards/JWT-SVID.md) does **not** require an `iss` claim, and several implementations omit it by default. **For this feature to work you must:**

- issue JWT-SVIDs that carry an **`iss` claim which is a URL serving OpenID Connect discovery + JWKS** that Artifactory can reach over the network, **and**
- configure the Artifactory OIDC provider with that **same issuer** URL.

How you turn this on is implementation-specific. For example, SPIRE exposes an [OIDC Discovery Provider](https://github.com/spiffe/spire/blob/main/support/oidc-discovery-provider/README.md) that serves the discovery document and JWKS for the trust domain; other implementations offer an equivalent OIDC endpoint. Consult your implementation's docs and see [spiffe.io](https://spiffe.io).

> **💡 Tip:** Fetch a JWT-SVID (see [Step 4 verification](#-verification)) and decode it (e.g. paste into a JWT decoder or `cut -d. -f2 | base64 -d`). Confirm there is an `iss` claim and that `<iss>/.well-known/openid-configuration` returns JSON from where Artifactory runs.

### 🔄 How It Works

```mermaid
sequenceDiagram
    participant Pod
    participant Kubelet
    participant Plugin as Credential Provider
    participant WLAPI as SPIFFE Workload API<br/>(node-local socket)
    participant OIDC as SPIFFE OIDC Discovery<br/>(issuer + JWKS)
    participant Artifactory as JFrog Artifactory

    Pod->>Kubelet: Request image pull
    Kubelet->>Plugin: Execute plugin (image matches pattern)
    Note over Plugin: Runs on the host;<br/>opens the node's Workload API socket
    Plugin->>WLAPI: FetchJWTSVID(audience = spiffe_svid_audience)
    WLAPI-->>Plugin: JWT-SVID (sub = SPIFFE ID, aud, iss)
    Plugin->>Artifactory: Exchange JWT-SVID for registry token<br/>(Generic OpenID Connect)
    Artifactory->>OIDC: GET <iss>/.well-known/openid-configuration + JWKS
    OIDC-->>Artifactory: Discovery doc + signing keys
    Note over Artifactory: Verifies signature and claims<br/>(iss, sub, aud)
    Artifactory-->>Plugin: Short-lived registry token
    Plugin-->>Kubelet: Return credential (username, token)
    Kubelet->>Artifactory: Pull image using credential
    Artifactory-->>Kubelet: Image data
    Kubelet-->>Pod: Image available
```

**Key components:**
- **SPIFFE Workload API**: node-local socket that issues the JWT-SVID (provided by your SPIFFE implementation).
- **JWT-SVID**: short-lived JWT with `sub` = the SPIFFE ID of the credential provider, `aud` = your configured `spiffe_svid_audience`, and an OIDC-discoverable `iss`.
- **SPIFFE OIDC discovery endpoint**: serves the issuer discovery document + JWKS that Artifactory uses to validate the token.
- **Artifactory**: validates the JWT-SVID and issues a short-lived registry token.

---

## ✅ Prerequisites

Before you begin, ensure you have the following:

- A **SPIFFE implementation deployed on your cluster** exposing a Workload API socket on each node that runs the credential provider (e.g. SPIRE agent as a DaemonSet, or a vendor equivalent).
- An **OIDC discovery endpoint** for your SPIFFE trust domain that Artifactory can reach (see the requirement above).
- **Access to JFrog Artifactory** with admin permissions.
- **kubectl** configured to access your cluster.
- **Helm 3.x** (if using Helm deployment).

### 🔍 Verify Prerequisites

```bash
# Confirm your SPIFFE implementation is running on the nodes (SPIRE example)
kubectl get pods -A | grep -i spire   # or your vendor's components

# Confirm the Workload API socket path on the node (implementation-specific).
# SPIRE default example:
#   unix:///run/spire/agent-sockets/spire-agent.sock

# Confirm the OIDC discovery endpoint is reachable (replace with your issuer URL)
curl -s https://<your-spiffe-issuer>/.well-known/openid-configuration | jq

# Check Helm (if using)
helm version
```

---

## 🚀 Setup Process

1. **SPIFFE Workload API** — ensure a Workload API socket is available on the nodes.
2. **OIDC discovery / issuer** — ensure JWT-SVIDs carry an OIDC-discoverable `iss` (see the requirement above).
3. **JFrog Artifactory OIDC** — create a Generic OpenID Connect provider + identity mapping.
4. **Deploy Credential Provider** — deploy with SPIFFE values.

---

## Step 1: 🧩 SPIFFE Workload API on the node

Deploy (or confirm) your SPIFFE implementation so that each node scheduled to run the credential provider exposes a Workload API socket, and note its path — you will pass it as `spiffe_endpoint_socket`.

- The path is **implementation-specific**. SPIRE's agent default, for example, is `unix:///run/spire/agent-sockets/spire-agent.sock`.
- The credential provider runs as a **host process** (executed by the kubelet), so it accesses the socket directly on the node filesystem; you do **not** need to mount the socket into the DaemonSet pod.
- `spiffe_endpoint_socket` is optional — if omitted, `go-spiffe` falls back to the standard `SPIFFE_ENDPOINT_SOCKET` environment variable.

Make sure your SPIFFE implementation is configured to **attest and issue an identity to the credential-provider workload** (the host process the kubelet executes). The SPIFFE ID it is granted becomes the `sub` claim you map in Artifactory (Step 3).

## Step 2: 🪪 Ensure an OIDC-discoverable issuer

Follow the [requirement section above](#️-requirement-an-oidc-discoverable-issuer-read-this-first): enable your implementation's OIDC discovery so issued JWT-SVIDs carry an `iss` that serves `/.well-known/openid-configuration` + JWKS reachable by Artifactory. Record the issuer URL — you'll use it as both `issuer_url` and `token_issuer` when creating the Artifactory OIDC provider.

## Step 3: 🐸 JFrog Artifactory OIDC Configuration

Configure Artifactory to accept your SPIFFE JWT-SVIDs by creating an OIDC provider and an identity mapping.

For more information, see the [JFrog Artifactory OIDC Documentation](https://www.jfrog.com/confluence/display/JFROG/Access+Tokens#AccessTokens-OIDCIntegration).

### 🔑 Get Artifactory Admin Token

Create an admin access token in Artifactory under **Administration** → **Identity and Access** → **Access Tokens**.

```bash
ARTIFACTORY_URL="your-instance.jfrog.io"
ARTIFACTORY_ADMIN_TOKEN="your-admin-access-token"
ARTIFACTORY_USER="spiffe-user"                 # Existing Artifactory user to map to
OIDC_PROVIDER_NAME="spiffe-oidc-provider"      # Choose a name
SPIFFE_ISSUER_URL="https://<your-spiffe-issuer>"   # OIDC-discoverable issuer from Step 2
SPIFFE_SVID_AUDIENCE="artifactory"             # The aud you will request for the JWT-SVID
SPIFFE_ID="spiffe://<trust-domain>/<workload-path>"   # SPIFFE ID granted to the provider
```

### ➕ Create OIDC Provider in Artifactory

```bash
curl -X POST "https://$ARTIFACTORY_URL/access/api/v1/oidc" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARTIFACTORY_ADMIN_TOKEN" \
  -d "{
    \"name\": \"$OIDC_PROVIDER_NAME\",
    \"issuer_url\": \"$SPIFFE_ISSUER_URL\",
    \"description\": \"OIDC provider for SPIFFE JWT-SVIDs\",
    \"provider_type\": \"Generic OpenID Connect\",
    \"token_issuer\": \"$SPIFFE_ISSUER_URL\",
    \"use_default_proxy\": false
  }"
```

> **⚠️ Important:** `issuer_url` / `token_issuer` must **exactly match** the `iss` claim in the JWT-SVID, and the discovery document at `$SPIFFE_ISSUER_URL/.well-known/openid-configuration` must be reachable from Artifactory.

### 🗺️ Create Identity Mapping

The identity mapping tells Artifactory how to map a SPIFFE JWT-SVID to an Artifactory user. Match on the SPIFFE ID (`sub`), the issuer (`iss`), and the audience (`aud`).

> **⚠️ Important:** Ensure `expires_in` is longer than the `defaultCacheDuration` set in your Helm values.

```bash
curl -X POST "https://$ARTIFACTORY_URL/access/api/v1/oidc/$OIDC_PROVIDER_NAME/identity_mappings" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARTIFACTORY_ADMIN_TOKEN" \
  -d "{
    \"name\": \"spiffe-identity-mapping\",
    \"description\": \"SPIFFE JWT-SVID identity mapping\",
    \"claims\": {
      \"iss\": \"$SPIFFE_ISSUER_URL\",
      \"sub\": \"$SPIFFE_ID\",
      \"aud\": \"$SPIFFE_SVID_AUDIENCE\"
    },
    \"token_spec\": {
      \"username\": \"$ARTIFACTORY_USER\",
      \"scope\": \"applied-permissions/user\",
      \"audience\": \"*@*\",
      \"expires_in\": 18000
    },
    \"priority\": 1
  }"
```

<details>
<summary><strong>📝 Configuration Notes</strong></summary>

- `claims.iss` must match the JWT-SVID `iss` and the provider's `token_issuer`.
- `claims.sub` is the SPIFFE ID granted to the credential-provider workload (e.g. `spiffe://example.org/ns/jfrog/sa/credential-provider`). Inspect a fetched JWT-SVID to see the exact value.
- `claims.aud` must match the `spiffe_svid_audience` you configure in Helm (the audience requested for the JWT-SVID).
- `token_spec.audience` (`*@*` above) is the audience of the **resulting Artifactory token** — this is the separate `jfrog_token_audience` setting (defaults to `*@*`). Set it only if your identity mapping uses a non-wildcard value.
- `token_spec.username` must be an existing Artifactory user with permission to pull from your repositories.

</details>

### ✅ Verify OIDC Provider

```bash
curl -X GET "https://$ARTIFACTORY_URL/access/api/v1/oidc/$OIDC_PROVIDER_NAME" \
  -H "Authorization: Bearer $ARTIFACTORY_ADMIN_TOKEN" | jq
```

## Step 4: 🚀 Deploy Credential Provider

Deploy the credential provider using Helm.

### 📝 Prepare Values File

Start from [`examples/spiffe-values.yaml`](./examples/spiffe-values.yaml):

| Configuration Value | Description | Example |
|---------------------|-------------|---------|
| `artifactoryUrl` | Your JFrog Artifactory URL | `your-instance.jfrog.io` |
| `spiffe.spiffe_endpoint_socket` | SPIFFE Workload API socket (optional; falls back to `SPIFFE_ENDPOINT_SOCKET`) | `unix:///run/spire/agent-sockets/spire-agent.sock` |
| `spiffe.spiffe_svid_audience` | Audience requested for the JWT-SVID (`aud` claim); must match the identity mapping | `artifactory` |
| `spiffe.jfrog_oidc_provider_name` | Name of the OIDC provider in Artifactory | `spiffe-oidc-provider` |
| `spiffe.jfrog_token_audience` | Audience of the resulting Artifactory token (optional; defaults to `*@*`) | `*@*` |

### 📦 Install with Helm

```bash
helm repo add jfrog https://charts.jfrog.io
helm repo update

helm upgrade --install secret-provider jfrog/jfrog-credential-provider \
  --namespace jfrog \
  --create-namespace \
  -f ./examples/spiffe-values.yaml
```

---

## ✅ Verification

### 📊 Check DaemonSet Status

```bash
kubectl get daemonset -n jfrog
kubectl get pods -n jfrog
```

All pods should be in `Running` state.

### 🧪 Confirm a JWT-SVID can be fetched (and inspect its claims)

Use your implementation's tooling on a node to confirm a JWT-SVID is issued for your audience, then decode it to verify the `iss`, `sub`, and `aud` claims match your Artifactory mapping.

```bash
# SPIRE example — run on a node with the agent socket:
spire-agent api fetch jwt -audience artifactory \
  -socketPath /run/spire/agent-sockets/spire-agent.sock

# Decode the payload of a fetched token to inspect claims:
echo "<jwt>" | cut -d. -f2 | base64 -d 2>/dev/null | jq
# Expect: an "iss" that is OIDC-discoverable, "sub" = your SPIFFE ID, "aud" = your audience
```

### 🧪 Test Image Pull

```bash
kubectl get nodes

kubectl run test-pull \
  --image=your-instance.jfrog.io/your-repo/test-image:latest \
  --restart=Never \
  --rm -it \
  --overrides='{"spec":{"nodeName":"your-node-name"}}'

kubectl describe pod test-pull
```

---

## 🔧 Troubleshooting

- **`invalid issuer` / signature verification failed at Artifactory** — the JWT-SVID `iss` does not match the provider `token_issuer`, or Artifactory cannot reach `<iss>/.well-known/openid-configuration` + JWKS. Re-check [the issuer requirement](#️-requirement-an-oidc-discoverable-issuer-read-this-first).
- **No `iss` claim in the token** — your SPIFFE implementation is not configured for OIDC discovery. Enable it (e.g. SPIRE's OIDC Discovery Provider) so SVIDs are issued with an OIDC-discoverable issuer.
- **`failed to fetch JWT-SVID from SPIFFE Workload API`** — the socket path is wrong or the agent is not running on that node. Verify `spiffe_endpoint_socket` and that the socket exists on the host.
- **`no identity mapping` / access denied** — the `sub`/`aud` in the mapping do not match the fetched token. Decode the token and align the mapping claims.

For general debugging, see the [debug documentation](./debug.md).

---

## 📚 Additional Resources

- [SPIFFE — spiffe.io](https://spiffe.io)
- [SPIFFE JWT-SVID specification](https://github.com/spiffe/spiffe/blob/main/standards/JWT-SVID.md)
- [SPIFFE Workload API specification](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE_Workload_API.md)
- [go-spiffe library](https://github.com/spiffe/go-spiffe)
- [SPIRE OIDC Discovery Provider](https://github.com/spiffe/spire/blob/main/support/oidc-discovery-provider/README.md) (one implementation's OIDC endpoint)
- [JFrog Artifactory OIDC Documentation](https://www.jfrog.com/confluence/display/JFROG/Access+Tokens#AccessTokens-OIDCIntegration)
- [Kubernetes Kubelet Credential Provider](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider/)
- [Main README](./README.md)
