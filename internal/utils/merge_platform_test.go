package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// OpenShift merges the JFrog provider into the platform kubelet credential config
// (ecr-credential-provider.yaml on AWS, acr-credential-provider.yaml on Azure).
// These tests ensure native provider fields (e.g. args) survive the merge.
func TestMergeOpenShiftPlatformPreservesNativeProvider(t *testing.T) {
	cases := []struct {
		name          string
		cloudProvider string
		platformFile  string
		platformYAML  string
		jfrogYAML     string
		mustContain   []string
	}{
		{
			name:          "aws_ecr",
			cloudProvider: CloudProviderAWS,
			platformFile:  "ecr-credential-provider.yaml",
			platformYAML: `apiVersion: kubelet.config.k8s.io/v1
kind: CredentialProviderConfig
providers:
  - name: ecr-credential-provider
    apiVersion: credentialprovider.kubelet.k8s.io/v1
    defaultCacheDuration: "4h"
    matchImages:
      - "*.dkr.ecr.*.amazonaws.com"
    args:
      - /etc/kubernetes/cloud.conf
`,
			jfrogYAML: `name: jfrog-credentials-provider
apiVersion: credentialprovider.kubelet.k8s.io/v1
matchImages:
  - "example.jfrog.io"
defaultCacheDuration: "5h"
tokenAttributes:
  serviceAccountTokenAudience: sts.amazonaws.com
  cacheType: ServiceAccount
  requireServiceAccount: true
  requiredServiceAccountAnnotationKeys:
    - eks.amazonaws.com/role-arn
    - JFrogExchange
env:
  - name: artifactory_url
    value: example.jfrog.io
  - name: aws_auth_method
    value: assume_role
  - name: aws_region
    value: us-east-1
`,
			mustContain: []string{
				"args:",
				"/etc/kubernetes/cloud.conf",
				"ecr-credential-provider",
				"jfrog-credentials-provider",
				"sts.amazonaws.com",
			},
		},
		{
			name:          "azure_acr",
			cloudProvider: CloudProviderAzure,
			platformFile:  "acr-credential-provider.yaml",
			platformYAML: `apiVersion: kubelet.config.k8s.io/v1
kind: CredentialProviderConfig
providers:
  - name: acr-credential-provider
    apiVersion: credentialprovider.kubelet.k8s.io/v1
    defaultCacheDuration: "10m"
    matchImages:
      - "*.azurecr.io"
    args:
      - /etc/kubernetes/cloud.conf
`,
			jfrogYAML: `name: jfrog-credentials-provider
apiVersion: credentialprovider.kubelet.k8s.io/v1
matchImages:
  - "example.jfrog.io"
defaultCacheDuration: "5h"
tokenAttributes:
  serviceAccountTokenAudience: api://AzureADTokenExchange
  cacheType: ServiceAccount
  requireServiceAccount: true
  requiredServiceAccountAnnotationKeys:
    - azure.workload.identity/client-id
    - JFrogExchange
env:
  - name: artifactory_url
    value: example.jfrog.io
  - name: azure_app_client_id
    value: test-client
  - name: azure_app_audience
    value: api://AzureADTokenExchange
  - name: jfrog_oidc_provider_name
    value: test-oidc
`,
			mustContain: []string{
				"args:",
				"/etc/kubernetes/cloud.conf",
				"acr-credential-provider",
				"jfrog-credentials-provider",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			platformPath := filepath.Join(dir, tc.platformFile)
			jfrogPath := filepath.Join(dir, "jfrog-provider.yaml")

			if err := os.WriteFile(platformPath, []byte(tc.platformYAML), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(jfrogPath, []byte(tc.jfrogYAML), 0644); err != nil {
				t.Fatal(err)
			}

			var config CredentialProviderConfig
			if err := ReadFile(platformPath, true, &config, tc.cloudProvider); err != nil {
				t.Fatal(err)
			}
			var provider Provider
			if err := ReadFile(jfrogPath, true, &provider, tc.cloudProvider); err != nil {
				t.Fatal(err)
			}
			config.Providers = append(config.Providers, provider)

			merged, err := yaml.Marshal(&config)
			if err != nil {
				t.Fatal(err)
			}
			mergedStr := string(merged)
			for _, sub := range tc.mustContain {
				if !strings.Contains(mergedStr, sub) {
					t.Fatalf("merged config missing %q:\n%s", sub, mergedStr)
				}
			}
		})
	}
}
