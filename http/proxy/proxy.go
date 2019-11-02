package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSainsburyPLC/ui-dev-proxy/domain"
	"github.com/JSainsburyPLC/ui-dev-proxy/http/rewrite"

	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const routeCtxKey = "route"

type Proxy struct {
	server      *http.Server
	TlsEnabled  bool
	TlsCertFile string
	TlsKeyFile  string
}

func NewProxy(
	port int,
	conf domain.Config,
	defaultBackend *url.URL,
	mocksEnabled bool,
	logger *log.Logger,
) *Proxy {
	reverseProxy := &httputil.ReverseProxy{
		Director:     director(defaultBackend, logger),
		ErrorHandler: errorHandler(logger),
	}
	return &Proxy{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: handler(logger, reverseProxy, conf, domain.NewMatcher(), mocksEnabled),
		},
	}
}

func (p *Proxy) Start() {
	if p.TlsEnabled {
		err := p.server.ListenAndServeTLS(p.TlsCertFile, p.TlsKeyFile)
		if err != nil {
			panic(err)
		}
		return
	}

	err := p.server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func director(defaultBackend *url.URL, logger *log.Logger) func(req *http.Request) {
	return func(req *http.Request) {
		route, ok := req.Context().Value(routeCtxKey).(*domain.Route)
		if !ok {
			// if not route set, then direct to default backend
			req.URL.Scheme = defaultBackend.Scheme
			req.URL.Host = defaultBackend.Host
			req.Host = defaultBackend.Host
			return
		}
		// if route is set redirect to route backend
		req.URL.Scheme = route.Backend.Scheme
		req.URL.Host = route.Backend.Host
		req.Host = route.Backend.Host

		// apply any defined rewrite rules
		for pattern, to := range route.Rewrite {
			rule, err := rewrite.NewRule(pattern, to)
			if err != nil {
				logger.Println(fmt.Sprintf("error creating rewrite rule. %v", err))
				continue
			}

			matched, err := rule.Rewrite(req)
			if err != nil {
				logger.Println(fmt.Sprintf("failed to rewrite request. %v", err))
				continue
			}

			// recursive rewrites are not supported, exit on first rewrite
			if matched {
				break
			}
		}
	}
}

func errorHandler(logger *log.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Printf("%+v\n", err)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("Bad gateway"))
	}
}

func handler(
	logger *log.Logger,
	reverseProxy *httputil.ReverseProxy,
	conf domain.Config,
	matcher domain.Matcher,
	mocksEnabled bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("inbound request on '%s %s'\n", r.Method, r.URL.String())

		matchedRoute, err := matchRoute(conf, matcher, r, mocksEnabled)
		if err != nil {
			logger.Printf(err.Error())
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("Bad gateway"))
		}

		if matchedRoute == nil {
			logger.Println("directing to default backend")
			reverseProxy.ServeHTTP(w, r)
			return
		}

		switch matchedRoute.Type {
		case domain.RouteTypeProxy:
			logger.Printf("directing to route backend '%s'\n", matchedRoute.Backend.Host)
			r = r.WithContext(context.WithValue(r.Context(), routeCtxKey, matchedRoute))
			reverseProxy.ServeHTTP(w, r)
		case domain.RouteTypeMock:
			if !mocksEnabled {
				logger.Println("directing to default backend")
				reverseProxy.ServeHTTP(w, r)
				return
			}
			logger.Printf("directing to mock: %+v\n", matchedRoute.Mock.Response)
			writeMockResponse(matchedRoute.Mock.Response, w)
		}
	}
}

func matchRoute(conf domain.Config, matcher domain.Matcher, r *http.Request, mocksEnabled bool) (*domain.Route, error) {
	for _, route := range conf.Routes {
		switch route.Type {
		case domain.RouteTypeProxy:
			if route.PathPattern.MatchString(r.URL.Path) {
				return &route, nil
			}
		case domain.RouteTypeMock:
			if mocksEnabled {
				if route.Mock == nil {
					return nil, errors.New("missing mock in config")
				}
				if matcher.Match(r, *route.Mock) {
					return &route, nil
				}
			}
		default:
			return nil, fmt.Errorf("unknown route type '%s'", route.Type)
		}
	}
	return nil, nil
}

func writeMockResponse(response domain.Response, w http.ResponseWriter) {
	body := []byte(response.Body)

	if json.Valid(body) {
		w.Header().Set("Content-Type", "application/json")
	}

	for _, cookie := range response.Cookies {
		addCookie(w, cookie)
	}

	w.WriteHeader(response.Status)
	_, _ = w.Write(body)
}

func addCookie(w http.ResponseWriter, cookie domain.Cookie) {
	expire := time.Now().Add(time.Hour * 24 * 7 * 30)
	c := http.Cookie{
		Name:    cookie.Name,
		Value:   cookie.Value,
		Expires: expire,
		MaxAge:  cookie.MaxAge,
		Path:    "/",
	}
	http.SetCookie(w, &c)
}
