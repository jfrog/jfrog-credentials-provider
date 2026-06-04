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

package provider

import (
	"net/http"
	"time"
)

func newProviderHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.IdleConnTimeout = 10 * time.Second
	transport.DisableCompression = true
	transport.Proxy = http.ProxyFromEnvironment

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
