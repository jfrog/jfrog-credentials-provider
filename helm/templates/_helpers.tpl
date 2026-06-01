{{/*
Expand the name of the chart.
*/}}
{{- define "jfrog-credential.provider.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "jfrog-credential-provider.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label - uses jfrog-common helper
*/}}
{{- define "jfrog-credential-provider.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels - uses jfrog-common helper with custom labels support
*/}}
{{- define "jfrog-credential-provider.labels" -}}
{{- include "common.labels.standard" (dict "customLabels" .Values.labels "context" .) }}
{{- end }}

{{/*
Selector labels - uses jfrog-common helper
*/}}
{{- define "jfrog-credential-provider.selectorLabels" -}}
{{- include "common.labels.matchLabels" (dict "customLabels" .Values.labels "context" .) }}
{{- end }}


{{/*
Get init container image
*/}}
{{- define "jfrog-credential-provider.initContainerImage" -}}
{{- if .Values.initContainer.image.digest }}
{{- printf "%s/%s@%s" .Values.initContainer.image.registry .Values.initContainer.image.repository .Values.initContainer.image.digest }}
{{- else }}
{{- printf "%s/%s:%s" .Values.initContainer.image.registry .Values.initContainer.image.repository .Values.initContainer.image.tag }}
{{- end }}
{{- end }}

{{/*
Get pause container image
*/}}
{{- define "jfrog-credential-provider.pauseImage" -}}
{{- if .Values.image.digest }}
{{- printf "%s/%s@%s" .Values.image.registry .Values.image.repository .Values.image.digest }}
{{- else }}
{{- printf "%s/%s:%s" .Values.image.registry .Values.image.repository .Values.image.tag }}
{{- end }}
{{- end }}

{{/*
Get service account name
*/}}
{{- define "jfrog-credential-provider.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "jfrog-credential-provider.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get namespace - uses jfrog-common helper
*/}}
{{- define "jfrog-credential-provider.namespace" -}}
{{- default .Values.namespace (include "common.names.namespace" .) }}
{{- end }}


# Outputs the cloud provider type. Since only one cloud provider is supported per installation,
# this returns the cloudProvider as a string ("aws", "azure", "gcp"), or an empty string if none detected.
{{- define "jfrog-credential-provider.cloudProvider" -}}
{{- $cloudProvider := "" -}}
{{- if .Values.providerConfig }}
  {{- range .Values.providerConfig }}
    {{- if and .aws .aws.enabled }}{{- $cloudProvider = "aws" -}}{{- end }}
    {{- if and .azure .azure.enabled }}{{- $cloudProvider = "azure" -}}{{- end }}
    {{- if and .gcp .gcp.enabled }}{{- $cloudProvider = "gcp" -}}{{- end }}
  {{- end }}
{{- end }}
{{- $cloudProvider -}}
{{- end }}

{{/*
Default RBAC rules for AWS IRSA with service account token projection
*/}}
{{- define "jfrog-credential-provider.defaultRBACRulesAWS" -}}
- apiGroups: [""]
  resources: ["serviceaccounts"]
  verbs: ["get", "list"]
- apiGroups: [""]
  resources: ["sts.amazonaws.com"]
  verbs: ["request-serviceaccounts-token-audience"]
{{- end }}

{{/*
Default RBAC rules for azure with service account token projection
*/}}
{{- define "jfrog-credential-provider.defaultRBACRulesAzure" -}}
- apiGroups: [""]
  resources: ["api://AzureADTokenExchange"]
  verbs: ["request-serviceaccounts-token-audience"]
- apiGroups: [""]
  resources: ["serviceaccounts/token"]
  verbs: ["create"]
- apiGroups: [""]
  resources: ["serviceaccounts"]
  verbs: ["get", "list"]
{{- end }}


{{/*
Default RBAC rules for gcp with service account token projection
*/}}
{{- define "jfrog-credential-provider.defaultRBACRulesGcp" }}
- apiGroups: [""]
  resources:
    - {{ include "jfrog-credential-provider.jfrogGCPAudience" . | quote }}
  verbs:
    - request-serviceaccounts-token-audience
{{- end }}

{{/*
Fetching JFrog audience from values configuration
*/}}
{{- define "jfrog-credential-provider.jfrogGCPAudience" -}}
{{- $default := "artifactory" -}}
{{- $audience := $default -}}
{{- range .Values.providerConfig | default list }}
  {{- if and (.tokenAttributes) (.gcp) (.tokenAttributes.enabled) (.gcp.enabled) }}
    {{- $audience = (default $default .gcp.jfrog_oidc_audience) }}
    {{- break -}}
  {{- end }}
{{- end }}
{{- $audience -}}
{{- end }}

{{/*
True only when platform: openshift is set. Omitted platform uses legacy EKS/AKS/GKE paths.
*/}}
{{- define "jfrog-credential-provider.isOpenShift" -}}
{{- eq .Values.platform "openshift" -}}
{{- end }}

{{/*
True when the chart should manage Pod Security Admission labels on the release namespace.
*/}}
{{- define "jfrog-credential-provider.openshiftLabelNamespacePodSecurity" -}}
{{- if ne (include "jfrog-credential-provider.isOpenShift" .) "true" -}}
false
{{- else if not .Values.openshift.labelNamespacePodSecurity -}}
false
{{- else -}}
true
{{- end -}}
{{- end }}

{{/*
True when OpenShift AWS uses projected SA only (no aws_role_name node fallback in kubelet config).
*/}}
{{- define "jfrog-credential-provider.awsOmitRoleNameFallback" -}}
{{- $item := index . "item" -}}
{{- $root := index . "root" -}}
{{- if ne (include "jfrog-credential-provider.isOpenShift" $root) "true" -}}
false
{{- else if not (and $item.tokenAttributes $item.tokenAttributes.enabled $item.tokenAttributes.requireServiceAccount) -}}
false
{{- else -}}
true
{{- end -}}
{{- end }}

{{/*
OpenShift on AWS or Azure: RHCOS paths, staging dir, bind-mount over /usr/libexec.
*/}}
{{- define "jfrog-credential-provider.isOpenShiftStaging" -}}
{{- if not (eq (include "jfrog-credential-provider.isOpenShift" .) "true") -}}
false
{{- else -}}
{{- $cp := include "jfrog-credential-provider.cloudProvider" . -}}
{{- if or (eq $cp "aws") (eq $cp "azure") -}}
true
{{- else -}}
false
{{- end -}}
{{- end -}}
{{- end }}

{{/*
Built-in platform credential plugin binary on OpenShift (ECR / ACR).
*/}}
{{- define "jfrog-credential-provider.openshiftPlatformPlugin" -}}
{{- $cp := include "jfrog-credential-provider.cloudProvider" . -}}
{{- if eq $cp "azure" -}}
acr-credential-provider
{{- else if eq $cp "gcp" -}}
gcr-credential-provider
{{- else -}}
ecr-credential-provider
{{- end -}}
{{- end }}

{{/*
Kubelet credential provider config file on OpenShift (targetProviderConfigDir + platform plugin name).
*/}}
{{- define "jfrog-credential-provider.openshiftKubeletConfigPath" -}}
{{- $dir := trimSuffix "/" .Values.openshift.targetProviderConfigDir -}}
{{ $dir }}/{{ include "jfrog-credential-provider.openshiftPlatformPlugin" . }}.yaml
{{- end }}

{{/*
True when kubelet credential provider config uses YAML (Azure, GCP, or OpenShift on AWS/Azure)
*/}}
{{- define "jfrog-credential-provider.kubeletConfigYaml" -}}
{{- $cloudProvider := include "jfrog-credential-provider.cloudProvider" . -}}
{{- if or (eq $cloudProvider "azure") (eq $cloudProvider "gcp") -}}
true
{{- else if and (eq $cloudProvider "aws") (eq (include "jfrog-credential-provider.isOpenShift" .) "true") -}}
true
{{- else -}}
false
{{- end -}}
{{- end }}

{{/*
DaemonSet needs host filesystem mount (not AKS-style /var/lib/kubelet only).
*/}}
{{- define "jfrog-credential-provider.useHostMount" -}}
{{- $cp := include "jfrog-credential-provider.cloudProvider" . -}}
{{- if or (eq $cp "aws") (eq $cp "gcp") (eq (include "jfrog-credential-provider.isOpenShiftStaging" .) "true") -}}
true
{{- else -}}
false
{{- end -}}
{{- end }}

{{/*
Env assignments for nsenter merge: MergeConfig probes cloud metadata from pod network unless cloud_provider is set.
OpenShift DaemonSet pods often cannot reach cloud instance metadata → force cloud from Helm.
GCP values use "gcp"; Go validates as "google".
*/}}
{{- define "jfrog-credential-provider.addProviderConfigEnvAssignments" -}}
{{- $cp := include "jfrog-credential-provider.cloudProvider" . -}}
{{- if eq $cp "aws" -}}
cloud_provider=aws
{{- else if eq $cp "azure" -}}
cloud_provider=azure
{{- else if eq $cp "gcp" -}}
cloud_provider=google
{{- end -}}
{{- end }}

{{/*
Azure env for YAML kubelet config. Omit empty tenant/nodepool IDs (workload identity does not need them).
*/}}
{{- define "jfrog-credential-provider.azureEnvYaml" -}}
{{- $item := .item -}}
{{- $values := .Values -}}
env:
  - name: artifactory_url
    value: {{ $item.artifactoryUrl | quote }}
  - name: azure_app_client_id
    value: {{ $item.azure.azure_app_client_id | quote }}
  {{- if $item.azure.azure_cloud_name }}
  - name: azure_cloud_name
    value: {{ $item.azure.azure_cloud_name | quote }}
  {{- end }}
  {{- if $item.azure.azure_tenant_id }}
  - name: azure_tenant_id
    value: {{ $item.azure.azure_tenant_id | quote }}
  {{- end }}
  {{- if $item.azure.azure_nodepool_client_id }}
  - name: azure_nodepool_client_id
    value: {{ $item.azure.azure_nodepool_client_id | quote }}
  {{- end }}
  - name: azure_app_audience
    value: {{ $item.azure.azure_app_audience | quote }}
  - name: jfrog_oidc_provider_name
    value: {{ $item.azure.jfrog_oidc_provider_name | quote }}
  - name: disable_provider_autoupdate
    value: {{ not $values.autoUpgrade | quote }}
  - name: log_level
    value: {{ $values.logLevel | quote }}
  {{- if $item.http_timeout_seconds }}
  - name: http_timeout_seconds
    value: {{ $item.http_timeout_seconds | quote }}
  {{- end }}
{{- end }}
