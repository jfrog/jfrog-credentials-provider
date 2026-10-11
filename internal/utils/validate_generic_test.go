// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import "testing"

func genericProvider(env ...EnvVar) Provider {
	return Provider{
		Name:                 "jfrog-credentials-provider",
		MatchImages:          []string{"example.jfrog.io"},
		DefaultCacheDuration: "5h",
		APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
		Env:                  env,
	}
}

// cloud_provider=generic is validated by the same switch that guards the AWS,
// Azure and Google paths, so the generic case needs the same coverage: the two
// OIDC settings it cannot run without must be rejected when absent.
func TestValidateJfrogProviderConfig_Generic(t *testing.T) {
	cases := []struct {
		name      string
		env       []EnvVar
		wantError bool
	}{
		{
			name: "valid",
			env: []EnvVar{
				{Name: "artifactory_url", Value: "example.jfrog.io"},
				{Name: "jfrog_oidc_provider_name", Value: "my-keycloak-provider"},
				{Name: "jfrog_oidc_audience", Value: "jfrog-artifactory-poc"},
			},
		},
		{
			name: "missing_provider_name",
			env: []EnvVar{
				{Name: "artifactory_url", Value: "example.jfrog.io"},
				{Name: "jfrog_oidc_audience", Value: "jfrog-artifactory-poc"},
			},
			wantError: true,
		},
		{
			name: "missing_audience",
			env: []EnvVar{
				{Name: "artifactory_url", Value: "example.jfrog.io"},
				{Name: "jfrog_oidc_provider_name", Value: "my-keycloak-provider"},
			},
			wantError: true,
		},
		{
			name: "missing_artifactory_url",
			env: []EnvVar{
				{Name: "jfrog_oidc_provider_name", Value: "my-keycloak-provider"},
				{Name: "jfrog_oidc_audience", Value: "jfrog-artifactory-poc"},
			},
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJfrogProviderConfig(genericProvider(tc.env...), CloudProviderGeneric)
			if tc.wantError && err == nil {
				t.Fatal("expected a validation error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected no validation error, got: %v", err)
			}
		})
	}
}
