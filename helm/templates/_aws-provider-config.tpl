{{/*
AWS provider tokenAttributes for YAML kubelet config (OpenShift / ROSA)
*/}}
{{- define "jfrog-credential-provider.awsTokenAttributesYaml" -}}
{{- if and .tokenAttributes .tokenAttributes.enabled }}
tokenAttributes:
  serviceAccountTokenAudience: sts.amazonaws.com
  cacheType: ServiceAccount
  requireServiceAccount: true
  {{- if .tokenAttributes.requireServiceAccount }}
  requiredServiceAccountAnnotationKeys:
    - eks.amazonaws.com/role-arn
    - JFrogExchange
  {{- else }}
  optionalServiceAccountAnnotationKeys:
    - eks.amazonaws.com/role-arn
    - JFrogExchange
  {{- end }}
{{- end }}
{{- end }}

{{/*
AWS provider env vars for YAML kubelet config (OpenShift / ROSA)
Context: dict with keys "item" (providerConfig entry) and "Values" (chart values)
*/}}
{{- define "jfrog-credential-provider.awsEnvYaml" -}}
{{- $item := .item -}}
{{- $values := .Values -}}
{{- $root := .Root -}}
env:
  - name: cloud_provider
    value: "aws"
  - name: aws_auth_method
    value: {{ $item.aws.aws_auth_method | quote }}
  - name: artifactory_url
    value: {{ $item.artifactoryUrl | quote }}
  - name: disable_provider_autoupdate
    value: {{ not $values.autoUpgrade | quote }}
  - name: log_level
    value: {{ $values.logLevel | quote }}
  {{- if $item.http_timeout_seconds }}
  - name: http_timeout_seconds
    value: {{ $item.http_timeout_seconds | quote }}
  {{- end }}
  {{- if $item.aws.aws_region }}
  - name: aws_region
    value: {{ $item.aws.aws_region | quote }}
  {{- end }}
  {{- if eq $item.aws.aws_auth_method "cognito_oidc" }}
  - name: secret_name
    value: {{ $item.aws.aws_cognito_user_pool_secret_name | quote }}
  - name: user_pool_name
    value: {{ $item.aws.aws_cognito_user_pool_name | quote }}
  - name: resource_server_name
    value: {{ $item.aws.aws_cognito_resource_server_name | quote }}
  - name: user_pool_resource_scope
    value: {{ $item.aws.aws_cognito_user_pool_resource_scope | quote }}
  - name: jfrog_oidc_provider_name
    value: {{ $item.aws.jfrog_oidc_provider_name | quote }}
  {{- else if eq $item.aws.aws_auth_method "assume_role" }}
  {{- if ne (include "jfrog-credential-provider.awsOmitRoleNameFallback" (dict "item" $item "root" $root)) "true" }}
  - name: aws_role_name
    value: {{ $item.aws.aws_role_name | quote }}
  {{- end }}
  - name: secret_ttl_seconds
    value: {{ ($item.aws.secret_ttl_seconds | default 14400) | quote }}
  {{- end }}
{{- end }}
