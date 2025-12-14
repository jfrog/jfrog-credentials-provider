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

