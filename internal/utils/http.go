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
	for k, v := range request.Header {
		req.Header.Add(k, v[0])
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
