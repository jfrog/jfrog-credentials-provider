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
	"context"
	"fmt"
	service "jfrog-credential-provider/internal"
	"os"
	"strings"

	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

// spiffeSocketAddress resolves the SPIFFE Workload API socket address from the
// provider config (spiffe_endpoint_socket) or, as a fallback, the standard
// SPIFFE_ENDPOINT_SOCKET environment variable read directly by the SDK.
func spiffeSocketAddress() string {
	if socket := os.Getenv("spiffe_endpoint_socket"); socket != "" {
		return socket
	}
	return os.Getenv("SPIFFE_ENDPOINT_SOCKET")
}

// GetSpiffeJWTSVID fetches a short-lived JWT-SVID for the given audience from the
// local SPIFFE Workload API endpoint. The endpoint is provided by whatever SPIFFE
// implementation the node runs (see https://spiffe.io); this code targets the
// SPIFFE Workload API itself, so it is implementation-agnostic.
//
// When socketPath is empty the go-spiffe client falls back to the standard
// SPIFFE_ENDPOINT_SOCKET environment variable.
func GetSpiffeJWTSVID(s *service.Service, ctx context.Context,
	socketPath string, audience string) (string, error) {
	s.Logger.Info("Fetching JWT-SVID from SPIFFE Workload API, socket: " + socketPath + ", audience: " + audience)

	var opts []workloadapi.ClientOption
	if socketPath != "" {
		opts = append(opts, workloadapi.WithAddr(socketPath))
	}

	svid, err := workloadapi.FetchJWTSVID(ctx, jwtsvid.Params{Audience: audience}, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to fetch JWT-SVID from SPIFFE Workload API: %v", err)
	}

	s.Logger.Info("Fetched JWT-SVID for SPIFFE ID: " + svid.ID.String())
	return svid.Marshal(), nil
}

// CheckIfSpiffe reports whether a SPIFFE Workload API socket is configured and
// present on the node. It is a cheap, local check (no network / gRPC dial): the
// socket address must be set (spiffe_endpoint_socket or SPIFFE_ENDPOINT_SOCKET)
// and the referenced unix socket file must exist.
func CheckIfSpiffe(s *service.Service, ctx context.Context) (bool, error) {
	s.Logger.Info("Checking if a SPIFFE Workload API socket is available")
	socket := spiffeSocketAddress()
	if socket == "" {
		return false, nil
	}

	// The address is typically a unix:// URL; stat the underlying socket file.
	socketFile := strings.TrimPrefix(socket, "unix://")
	if _, err := os.Stat(socketFile); err != nil {
		s.Logger.Info("SPIFFE endpoint socket not present at " + socketFile + ": " + err.Error())
		return false, nil
	}
	return true, nil
}
