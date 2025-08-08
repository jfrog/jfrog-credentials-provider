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

import (
	"bytes"
	"context"
	"fmt"
	"io"
	service "jfrog-credential-provider/internal"
	"net/http"
)

func HttpReq(s *service.Service, ctx context.Context, url string, body []byte, request *http.Request) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	// set headers
	req.Header.Set("Content-Type", "application/json")
	if request != nil {
		for k, v := range request.Header {
			req.Header.Add(k, v[0])
		}
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending artifactory create token request, Cause %s", err)
	}
	// defer func(Body io.ReadCloser) {
	// 	err := Body.Close()
	// 	if err != nil {
	// 		s.Logger.Error("Could not close response body" + err.Error())
	// 	}
	// }(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.Logger.Error("could not parse artifactory body " + err.Error())
		}
		errMessage := fmt.Sprintf("%s%s%s%d%s%s%s", "Error getting artifactory token request, token creation to ", url, " returned ", resp.StatusCode, " response", "body ", body)
		return nil, fmt.Errorf("message: %s", errMessage)
	}
	return resp, nil
}
