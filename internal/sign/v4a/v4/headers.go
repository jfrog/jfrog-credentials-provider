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

// IgnoredHeaders is a list of headers that are ignored during signing
var IgnoredHeaders = Rules{
	DenyList{
		MapRule{
			"Authorization":   struct{}{},
			"User-Agent":      struct{}{},
			"X-Amzn-Trace-Id": struct{}{},
		},
	},
}

// RequiredSignedHeaders is a whitelist for Build canonical headers.
var RequiredSignedHeaders = Rules{
	AllowList{
		MapRule{
			"Cache-Control":                         struct{}{},
			"Content-Disposition":                   struct{}{},
			"Content-Encoding":                      struct{}{},
			"Content-Language":                      struct{}{},
			"Content-Md5":                           struct{}{},
			"Content-Type":                          struct{}{},
			"Expires":                               struct{}{},
			"If-Match":                              struct{}{},
			"If-Modified-Since":                     struct{}{},
			"If-None-Match":                         struct{}{},
			"If-Unmodified-Since":                   struct{}{},
			"Range":                                 struct{}{},
			"X-Amz-Acl":                             struct{}{},
			"X-Amz-Copy-Source":                     struct{}{},
			"X-Amz-Copy-Source-If-Match":            struct{}{},
			"X-Amz-Copy-Source-If-Modified-Since":   struct{}{},
			"X-Amz-Copy-Source-If-None-Match":       struct{}{},
			"X-Amz-Copy-Source-If-Unmodified-Since": struct{}{},
			"X-Amz-Copy-Source-Range":               struct{}{},
			"X-Amz-Copy-Source-Server-Side-Encryption-Customer-Algorithm": struct{}{},
			"X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key":       struct{}{},
			"X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key-Md5":   struct{}{},
			"X-Amz-Grant-Full-control":                                    struct{}{},
			"X-Amz-Grant-Read":                                            struct{}{},
			"X-Amz-Grant-Read-Acp":                                        struct{}{},
			"X-Amz-Grant-Write":                                           struct{}{},
			"X-Amz-Grant-Write-Acp":                                       struct{}{},
			"X-Amz-Metadata-Directive":                                    struct{}{},
			"X-Amz-Mfa":                                                   struct{}{},
			"X-Amz-Request-Payer":                                         struct{}{},
			"X-Amz-Server-Side-Encryption":                                struct{}{},
			"X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id":                 struct{}{},
			"X-Amz-Server-Side-Encryption-Customer-Algorithm":             struct{}{},
			"X-Amz-Server-Side-Encryption-Customer-Key":                   struct{}{},
			"X-Amz-Server-Side-Encryption-Customer-Key-Md5":               struct{}{},
			"X-Amz-Storage-Class":                                         struct{}{},
			"X-Amz-Website-Redirect-Location":                             struct{}{},
			"X-Amz-Content-Sha256":                                        struct{}{},
			"X-Amz-Tagging":                                               struct{}{},
		},
	},
	Patterns{"X-Amz-Meta-"},
}

// AllowedQueryHoisting is a whitelist for Build query headers. The boolean value
// represents whether or not it is a pattern.
var AllowedQueryHoisting = InclusiveRules{
	DenyList{RequiredSignedHeaders},
	Patterns{"X-Amz-"},
}
