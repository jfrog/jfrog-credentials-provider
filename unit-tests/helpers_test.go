package unittests

import (
	"encoding/json"
	"fmt"
	"io"
	service "jfrog-credential-provider/internal"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/logger"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type routedService struct {
	service     *service.Service
	imdsServer  *httptest.Server
	jfrogServer *httptest.Server
}

func newRoutedService(t *testing.T, imdsHandler, jfrogHandler http.Handler) routedService {
	t.Helper()

	imdsServer := httptest.NewServer(imdsHandler)
	jfrogServer := httptest.NewTLSServer(jfrogHandler)
	t.Cleanup(imdsServer.Close)
	t.Cleanup(jfrogServer.Close)

	imdsURL, err := url.Parse(imdsServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	jfrogURL, err := url.Parse(jfrogServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		switch req.URL.Hostname() {
		case "169.254.169.254":
			cloned.URL.Scheme = imdsURL.Scheme
			cloned.URL.Host = imdsURL.Host
			return http.DefaultTransport.RoundTrip(cloned)
		case "artifactory.test":
			cloned.URL.Scheme = jfrogURL.Scheme
			cloned.URL.Host = jfrogURL.Host
			return jfrogServer.Client().Transport.RoundTrip(cloned)
		default:
			if strings.HasSuffix(req.URL.Hostname(), ".amazoncognito.com") {
				cloned.URL.Scheme = jfrogURL.Scheme
				cloned.URL.Host = jfrogURL.Host
				return jfrogServer.Client().Transport.RoundTrip(cloned)
			}
			return http.DefaultTransport.RoundTrip(cloned)
		}
	})

	testLogger := logger.Logger{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return routedService{
		service:     service.NewService(&http.Client{Transport: transport}, testLogger),
		imdsServer:  imdsServer,
		jfrogServer: jfrogServer,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMultiRoutedService(t *testing.T, handlersByHost map[string]http.Handler) *service.Service {
	t.Helper()

	type route struct {
		target    *url.URL
		transport http.RoundTripper
	}
	routes := make(map[string]route, len(handlersByHost))
	for host, handler := range handlersByHost {
		var server *httptest.Server
		if host == "169.254.169.254" {
			server = httptest.NewServer(handler)
		} else {
			server = httptest.NewTLSServer(handler)
		}
		t.Cleanup(server.Close)
		target, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		routes[host] = route{target: target, transport: server.Client().Transport}
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		route, ok := routes[req.URL.Hostname()]
		if !ok {
			return nil, fmt.Errorf("unit test blocked unexpected network host %q", req.URL.Hostname())
		}
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = route.target.Scheme
		cloned.URL.Host = route.target.Host
		return route.transport.RoundTrip(cloned)
	})
	testLogger := logger.Logger{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return service.NewService(&http.Client{Transport: transport}, testLogger)
}

func oidcExchangeHandler(t *testing.T, subjectToken, providerName, audience string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != handlers.OIDC_ENDPOINT {
			t.Errorf("unexpected JFrog OIDC exchange request: %s %s", r.Method, r.URL.Path)
		}
		var request handlers.OidcTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode JFrog OIDC exchange request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if request.SubjectToken != subjectToken {
			t.Errorf("unexpected subject token: %q", request.SubjectToken)
		}
		if request.ProviderName != providerName {
			t.Errorf("unexpected provider name: %q", request.ProviderName)
		}
		if request.Audience != audience {
			t.Errorf("unexpected audience: %q", request.Audience)
		}
		if request.GrantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
			t.Errorf("unexpected grant type: %q", request.GrantType)
		}
		_ = json.NewEncoder(w).Encode(handlers.OidcAccessResponse{
			Username:    "cloud-user",
			AccessToken: "jfrog-cloud-token",
		})
	})
}
