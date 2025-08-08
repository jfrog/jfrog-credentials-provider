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

package v4

const (
	// EmptyStringSHA256 is the hex encoded sha256 value of an empty string
	EmptyStringSHA256 = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

	// UnsignedPayload indicates that the request payload body is unsigned
	UnsignedPayload = "UNSIGNED-PAYLOAD"

	// AmzAlgorithmKey indicates the signing algorithm
	AmzAlgorithmKey = "X-Amz-Algorithm"

	// AmzSecurityTokenKey indicates the security token to be used with temporary credentials
	AmzSecurityTokenKey = "X-Amz-Security-Token"

	// AmzDateKey is the UTC timestamp for the request in the format YYYYMMDD'T'HHMMSS'Z'
	AmzDateKey = "X-Amz-Date"

	// AmzCredentialKey is the access key ID and credential scope
	AmzCredentialKey = "X-Amz-Credential"

	// AmzSignedHeadersKey is the set of headers signed for the request
	AmzSignedHeadersKey = "X-Amz-SignedHeaders"

	// AmzSignatureKey is the query parameter to store the SigV4 signature
	AmzSignatureKey = "X-Amz-Signature"

	// TimeFormat is the time format to be used in the X-Amz-Date header or query parameter
	TimeFormat = "20060102T150405Z"

	// ShortTimeFormat is the shorten time format used in the credential scope
	ShortTimeFormat = "20060102"

	// ContentSHAKey is the SHA256 of request body
	ContentSHAKey = "X-Amz-Content-Sha256"
)
