package proxy

import (
	"github.com/JSainsburyPLC/ui-dev-proxy/domain"
	"github.com/steinfletcher/apitest"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"testing"
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
		Body(`{"user_id": "123"}`).
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
	newApiTest(config(), "http://test-backend", true).
		Get("/api/users/info").
		Query("include", "user_id").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"user_id": "123456"}`).
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
