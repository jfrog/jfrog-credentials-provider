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

package handlers

import (
	"fmt"
	"jfrog-credential-provider/internal/utils"
)

// GetGenericIdentityToken returns the id_token for a generic / Keycloak OIDC
// login flow. Unlike AWS/Azure/Google, there is no cloud metadata service to
// call: kubelet already injects a bound, audience-scoped id_token into the
// CredentialProviderRequest (via tokenAttributes.serviceAccountTokenAudience
// in the provider config), so this just extracts it.
func GetGenericIdentityToken(request utils.CredentialProviderRequest) (string, error) {
	if request.ServiceAccountToken == "" {
		return "", fmt.Errorf("no serviceAccountToken present in CredentialProviderRequest; " +
			"set tokenAttributes.serviceAccountTokenAudience in the provider config for generic OIDC mode")
	}
	return request.ServiceAccountToken, nil
}
