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

// spiffeProvider builds a minimal, valid Provider for cloud_provider=spiffe with
// the given env vars merged over the required defaults.
func spiffeProvider(env []EnvVar) Provider {
	return Provider{
		Name:                 "jfrog-credentials-provider",
		MatchImages:          []string{"*.jfrog.io"},
		DefaultCacheDuration: "5h",
		Env:                  env,
	}
}

func TestValidateJfrogProviderConfigSpiffe(t *testing.T) {
	cases := []struct {
		name    string
		env     []EnvVar
		wantErr bool
	}{
		{
			name: "valid",
			env: []EnvVar{
				{Name: "artifactory_url", Value: "example.jfrog.io"},
				{Name: "spiffe_svid_audience", Value: "artifactory"},
				{Name: "jfrog_oidc_provider_name", Value: "spiffe-provider"},
			},
			wantErr: false,
		},
		{
			name: "missing_spiffe_svid_audience",
			env: []EnvVar{
				{Name: "artifactory_url", Value: "example.jfrog.io"},
				{Name: "jfrog_oidc_provider_name", Value: "spiffe-provider"},
			},
			wantErr: true,
		},
		{
			name: "missing_jfrog_oidc_provider_name",
			env: []EnvVar{
				{Name: "artifactory_url", Value: "example.jfrog.io"},
				{Name: "spiffe_svid_audience", Value: "artifactory"},
			},
			wantErr: true,
		},
		{
			name: "missing_artifactory_url",
			env: []EnvVar{
				{Name: "spiffe_svid_audience", Value: "artifactory"},
				{Name: "jfrog_oidc_provider_name", Value: "spiffe-provider"},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJfrogProviderConfig(spiffeProvider(tc.env), CloudProviderSpiffe)
			if tc.wantErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
