# JFrog Credential Provider Helm Chart - Changelog

All notable changes to this Helm chart will be documented in this file.

## [1.3.0] - 11th June, 2026

* Added optional `platform: openshift` for OpenShift on AWS or Azure — omit `platform` for EKS, AKS, and GKE; YAML merge into `ecr-credential-provider.yaml` (AWS) or `acr-credential-provider.yaml` (Azure)
* OpenShift projected SA (`tokenAttributes`): **OpenShift 4.21+** only; 4.20 kubelet rejects `tokenAttributes` without `KubeletServiceAccountTokenForCredentialProviders`
* Added [OpenShift.md](../OpenShift.md) (consolidates ROSA.md and ARO.md)
* Added [examples/openshift-azure-projected-sa-values.yaml](../examples/openshift-azure-projected-sa-values.yaml) and [examples/openshift-aws-projected-sa-values.yaml](../examples/openshift-aws-projected-sa-values.yaml)
* Added `providerConfig[].tokenAttributes.requireServiceAccount` for AWS / IRSA
* Added `openshift.grantPrivilegedSCC` for SCC on OpenShift
* Added `openshift.labelNamespacePodSecurity` (default `true`) to apply privileged Pod Security Admission labels on the release namespace; set `false` to manage labels outside Helm
* OpenShift namespace labels: chart sets `enforce` and `scc.podSecurityLabelSync` only (not `audit`/`warn`); create the release namespace before `helm install` and do not use `--create-namespace` (conflicts with chart-managed `Namespace` resource)
* OpenShift: stage plugin binary on `/var/lib/jfrog-credential-provider/bin` and bind-mount over read-only `/usr/libexec/kubelet-image-credential-provider-plugins`
* OpenShift: inject `cloud_provider` for `add-provider-config` (`MergeConfig` probes IMDS/metadata from pod network, which often fails on ROSA)

## [1.2.1] - 11th June, 2026
* Added `internalBinaryHostPath` to support air-gapped / AMI-baked binaries by skipping the download
* Added `binaryDownload.auth` to support authenticated binary downloads from a private Artifactory repository

## [1.2.0] - 5th June, 2026
* Added a fix to honor proxy env vars for provider HTTP clients
* Add assume_external_role auth method for cross-account IAM role assumption

## [1.1.2] - 22nd May, 2026
* Added support for `containerLogging` for plugins logs in container stdout

## [1.1.1] - 21st April, 2026
* Added support for Azure China configured using `azure_cloud_name`

## [1.1.0] - 7th April, 2026
* Added KEP-4412 - Pod Level Identity Support For JFrog Artifactory on GCP
* Added support for `http_timeout_seconds` for HTTP calls
* Fixed `secret_ttl_seconds` in configmap to handle quotes
* Removed `host` header from AWS Signed requests to Artifactory to prevent from overriding host issues on webserver

## [1.0.1] - 25th Mar, 2026
* Added support for disabling auto-upgrade of binary through `autoUpgrade`
* Added support for `aws_region` for `assume_role` authentication method

## [1.0.0] - 23rd Feb, 2026
* Allow using an existing ServiceAccount when `serviceAccount.create=false`
* Fixed `defaultCacheDuration` for AWS
* Updated timeout to 60 seconds for tailing for logs in init-container
* Added automatic rollback incase of config issues causing kubelet restarts
* **Breaking Change** 
* Moved `initContainer.image.imagePullSecrets` and `image.imagePullSecrets` to top-level `imagePullSecrets` to align with K8s spec.
