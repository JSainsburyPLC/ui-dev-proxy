package proxy

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/JSainsburyPLC/ui-dev-proxy/domain"
	"github.com/steinfletcher/apitest"
)

func newApiTest(
	conf domain.Config,
	defaultBackend string,
	mocksEnabled bool,
) *apitest.APITest {
	u, err := url.Parse(defaultBackend)
	if err != nil {
		panic(err)
	}
	logger := log.New(ioutil.Discard, "", log.LstdFlags)
	p := NewProxy(8080, conf, u, mocksEnabled, logger)
	return apitest.New().Handler(p.server.Handler)
}

func TestProxy_DefaultBackend_Success(t *testing.T) {
	newApiTest(config(), "http://test-backend", false).
		Mocks(
			defaultBackendMock(http.StatusOK, `{"product_id": "123"}`),
		).
		Get("/original-ui/product").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"product_id": "123"}`).
		End()
}

func TestProxy_ProxyBackend_OtherProxy_Success(t *testing.T) {
	newApiTest(config(), "http://test-backend", false).
		Mocks(
			otherProxyMock(http.StatusOK, `{"product_id": "123"}`),
		).
		Get("/test-ui/product").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"product_id": "123"}`).
		End()
}

func TestProxy_ProxyBackend_UserProxy_Success(t *testing.T) {
	newApiTest(config(), "http://test-backend", false).
		Mocks(
			userProxyMock(http.StatusOK, `{"user_id": "123"}`),
		).
		Get("/test-ui/users/info").
		Expect(t).
		Status(http.StatusOK).
		Header("Cache-Control", "no-cache").
		Body(`{"user_id": "123"}`).
		End()
}

func TestProxy_ProxyBackend_ResponseReplacements(t *testing.T) {
	backendMock := apitest.NewMock().Get("http://localhost:3001/test-ui/users/info").
		RespondWith().
		Status(http.StatusOK).
		Header("test-header", "test-value-1").
		Body(`{"product_id": "test-value-1"}`).
		End()

	newApiTest(config(), "http://test-backend", false).
		Mocks(backendMock).
		Get("/test-ui/users/info").
		Expect(t).
		Status(http.StatusOK).
		Header("test-header", "test-value-2").
		Body(`{"product_id": "test-value-2"}`).
		End()
}

func TestProxy_Rewrite(t *testing.T) {
	tests := map[string]struct {
		pattern string
		to      string
		before  string
		after   string
	}{
		"constant": {
			pattern: "/test-ui/users/info",
			to:      "/rewrite-ui/users/info",
			before:  "/test-ui/users/info",
			after:   "/rewrite-ui/users/info",
		},
		"preserves original URL if no match": {
			pattern: "^/other-ui/(.*)",
			to:      "/rewrite-ui/$1",
			before:  "/test-ui/users/info",
			after:   "/test-ui/users/info",
		},
		"match group": {
			pattern: "^/test-ui/(.*)",
			to:      "/rewrite-ui/$1",
			before:  "/test-ui/users/info",
			after:   "/rewrite-ui/users/info",
		},
		"multiple match groups": {
			pattern: "^/test-(.*)/users/(.*)",
			to:      "/rewrite-$1/$2",
			before:  "/test-ui/users/info",
			after:   "/rewrite-ui/info",
		},
		"encoded characters": {
			pattern: "^/test-ui/users/(.*)",
			to:      "/rewrite-ui/$1",
			before:  "/test-ui/users/x-1%2F",
			after:   "/rewrite-ui/x-1%2F",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockProxyUrlUserUi, _ := url.Parse("http://localhost:3001")
			route := domain.Route{
				Type:        "proxy",
				PathPattern: &domain.PathPattern{Regexp: regexp.MustCompile("^/test-ui/users/.*")},
				Backend:     &domain.Backend{URL: mockProxyUrlUserUi},
				Rewrite: []domain.Rewrite{{
					PathPattern: &domain.PathPattern{Regexp: regexp.MustCompile(test.pattern)},
					To:          test.to,
				}},
			}

			newApiTest(configWithRoutes(route), "http://test-backend", false).
				Mocks(apitest.NewMock().
					Get("http://localhost:3001" + test.after).
					RespondWith().
					Status(http.StatusOK).
					Body(`{"user_id": "123"}`).
					End()).
				Get(test.before).
				Expect(t).
				Status(http.StatusOK).
				Body(`{"user_id": "123"}`).
				End()
		})
	}
}

func TestProxy_ProxyBackend_Redirect_Temporary(t *testing.T) {
	route := domain.Route{
		Type:        "redirect",
		PathPattern: &domain.PathPattern{Regexp: regexp.MustCompile("/test-ui/(.*)")},
		Redirect: &domain.Redirect{
			To:   "http://www.domain2.com/redirect-ui/$1",
			Type: "temporary",
		},
	}

	newApiTest(configWithRoutes(route), "http://test-backend", false).
		Get("/test-ui/users/info").
		Expect(t).
		Status(http.StatusFound).
		Header("Location", "http://www.domain2.com/redirect-ui/users/info").
		End()
}

func TestProxy_ProxyBackend_Redirect_Permanent(t *testing.T) {
	route := domain.Route{
		Type:        "redirect",
		PathPattern: &domain.PathPattern{Regexp: regexp.MustCompile("/test-ui/(.*)")},
		Redirect: &domain.Redirect{
			To:   "http://www.domain2.com/redirect-ui/$1",
			Type: "permanent",
		},
	}

	newApiTest(configWithRoutes(route), "http://test-backend", false).
		Get("/test-ui/users/info").
		Expect(t).
		Status(http.StatusMovedPermanently).
		Header("Location", "http://www.domain2.com/redirect-ui/users/info").
		End()
}

func TestProxy_MockBackend_Failure(t *testing.T) {
	newApiTest(config(), "http://test-backend", false).
		Mocks(
			mockBackendMock(http.StatusOK, `{"user_id": "123"}`),
		).
		Get("/api/users/info").
		Query("include", "user_id").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"user_id": "123"}`).
		End()
}

func TestProxy_MocksEnabled_DefaultBackend_Success(t *testing.T) {
	newApiTest(config(), "http://test-backend", true).
		Mocks(
			defaultBackendMock(http.StatusOK, `{"product_id": "123"}`),
		).
		Get("/original-ui/product").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"product_id": "123"}`).
		End()
}

func TestProxy_MocksEnabled_ProxyBackend_Success(t *testing.T) {
	newApiTest(config(), "http://test-backend", true).
		Mocks(
			otherProxyMock(http.StatusOK, `{"product_id": "123"}`),
		).
		Get("/test-ui/product").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"product_id": "123"}`).
		End()
}

func TestProxy_MocksEnabled_MockBackend_Success(t *testing.T) {
	conf := config()
	conf.Routes = []domain.Route{
		{
			Type: "mock",
			Mock: &domain.Mock{
				MatchRequest: domain.MatchRequest{
					Method: "GET",
					Path:   "^/api/users/.*",
					Query:  "c=3",
				},
				Response: domain.Response{
					Status: 200,
					Body:   `{"name": "bob"}`,
				},
			},
		},
		{
			Type: "mock",
			Mock: &domain.Mock{
				MatchRequest: domain.MatchRequest{
					Method: "GET",
					Path:   "^/api/users/.*",
					Query:  "a=1&b=2",
				},
				Response: domain.Response{
					Status: 200,
					Body:   `{"name": "jon"}`,
				},
			},
		},
	}

	newApiTest(conf, "http://test-backend", true).
		Intercept(func(request *http.Request) {
			request.URL.RawQuery = "a=1&b=2"
		}).
		Get("/api/users/info").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon"}`).
		End()

	newApiTest(conf, "http://test-backend", true).
		Intercept(func(request *http.Request) {
			request.URL.RawQuery = "b=2&a=1"
		}).
		Get("/api/users/info").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon"}`).
		End()

	newApiTest(conf, "http://test-backend", true).
		Intercept(func(request *http.Request) {
			request.URL.RawQuery = "c=3"
		}).
		Get("/api/users/info").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "bob"}`).
		End()

	newApiTest(conf, "http://test-backend", true).
		Get("/api/users/info").
		Expect(t).
		Status(http.StatusBadGateway).
		End()
}

func TestProxy_InvalidRouteType_Failure(t *testing.T) {
	newApiTest(invalidTypeConfig(), "http://test-backend", false).
		Get("/api/users/info").
		Expect(t).
		Status(http.StatusBadGateway).
		End()
}

func mockBackendMock(status int, responseBody string) *apitest.Mock {
	return apitest.NewMock().Get("http://test-backend/api/users/info").
		RespondWith().
		Status(status).
		Body(responseBody).
		End()
}

func otherProxyMock(status int, responseBody string) *apitest.Mock {
	return apitest.NewMock().Get("http://localhost:3002/test-ui/product").
		RespondWith().
		Status(status).
		Body(responseBody).
		End()
}

func userProxyMock(status int, responseBody string) *apitest.Mock {
	return apitest.NewMock().Get("http://localhost:3001/test-ui/users/info").
		Header("Referer", "https://www.test.example.com").
		RespondWith().
		Status(status).
		Body(responseBody).
		End()
}

func defaultBackendMock(status int, responseBody string) *apitest.Mock {
	return apitest.NewMock().Get("http://test-backend/original-ui/product").
		RespondWith().
		Status(status).
		Body(responseBody).
		End()
}

func config() domain.Config {
	mockProxyUrlUserUi, err := url.Parse("http://localhost:3001")
	if err != nil {
		panic(err)
	}
	mockProxyUrlOtherUi, err := url.Parse("http://localhost:3002")
	if err != nil {
		panic(err)
	}
	return domain.Config{
		Routes: []domain.Route{
			{
				Type:        "proxy",
				PathPattern: &domain.PathPattern{Regexp: regexp.MustCompile("^/test-ui/users/.*")},
				Backend:     &domain.Backend{URL: mockProxyUrlUserUi},
				ProxyPassHeaders: map[string]string{
					"Referer": "https://www.test.example.com",
				},
				ProxyResponseHeaders: map[string]string{
					"Cache-Control": "no-cache",
				},
				ProxyResponseReplacements: map[string]string{
					"test-value-1": "test-value-2",
				},
			},
			{
				Type:        "proxy",
				PathPattern: &domain.PathPattern{Regexp: regexp.MustCompile("^/test-ui/.*")},
				Backend:     &domain.Backend{URL: mockProxyUrlOtherUi},
			},
			{
				Type: "mock",
				Mock: &domain.Mock{
					MatchRequest: domain.MatchRequest{
						Method: "GET",
						Path:   "^/api/users/.*",
						Query:  "include=.*",
					},
					Response: domain.Response{
						Status: 200,
						Body:   `{"user_id": "123456"}`,
						Cookies: []domain.Cookie{
							{
								Name:   "SOME_COOKIE",
								Value:  "1234567890",
								MaxAge: 604800,
							},
						},
					},
				},
			},
		},
	}
}

func configWithRoutes(routes ...domain.Route) domain.Config {
	conf := config()
	conf.Routes = routes
	return conf
}

func invalidTypeConfig() domain.Config {
	mockProxyUrlUserUi, err := url.Parse("http://localhost:3001")
	if err != nil {
		panic(err)
	}
	return domain.Config{
		Routes: []domain.Route{
			{
				Type:        "not_a_proxy",
				PathPattern: &domain.PathPattern{Regexp: regexp.MustCompile("^/test-ui/users/.*")},
				Backend:     &domain.Backend{URL: mockProxyUrlUserUi},
			},
		},
	}
}
