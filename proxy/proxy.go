package proxy

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jackwakefield/gopac"

	"github.com/JSainsburyPLC/ui-dev-proxy/domain"
)

const routeCtxKey = "route"

type Proxy struct {
	server      *http.Server
	TlsEnabled  bool
	TlsCertFile string
	TlsKeyFile  string
}

func sainsHTTPTransport(logger *log.Logger) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
		DualStack: true,
	}
	parser := new(gopac.Parser)
	pacFile := os.Getenv("PAC_FILE")
	proxyFunc := http.ProxyFromEnvironment
	if pacFile != "" {
		pacUrl, err := url.Parse(pacFile)
		if err != nil {
			panic(err)
		}
		if pacUrl.Scheme == "file" {
			err = parser.Parse("/Users/kevin.cross/web-proxy5.pac")
		} else {
			err = parser.ParseUrl(pacFile)
		}
		if err != nil {
			panic(err)
		}

		m := sync.Mutex{}

		proxyFunc = func(request *http.Request) (*url.URL, error) {
			reqUrl := request.URL.String()
			reqHost := request.URL.Hostname()
			m.Lock()
			entry, err2 := findProxy(parser, reqUrl, reqHost)
			m.Unlock()
			if err2 != nil {
				logger.Printf("ERROR from pac: %s", err2)
				return nil, err2
			}
			if entry == "DIRECT" {
				return nil, nil
			}
			newEntry := strings.ToLower(entry)
			proxys := strings.Split(newEntry, ";")
			proxy := strings.TrimSpace(strings.TrimPrefix(proxys[0], "proxy"))
			return &url.URL{Host: proxy}, nil
		}
	}

	tr := &http.Transport{
		Proxy:                 proxyFunc,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       20 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
	}

	return tr
}
func findProxy(parser *gopac.Parser, reqUrl string, reqHost string) (entry string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
			entry = ""
		}
	}()
	return parser.FindProxy(reqUrl, reqHost)
}

func NewProxy(
	port int,
	conf domain.Config,
	defaultBackend *url.URL,
	mocksEnabled bool,
	logger *log.Logger,
) *Proxy {
	reverseProxy := &httputil.ReverseProxy{
		Director:       director(defaultBackend, logger),
		Transport:      sainsHTTPTransport(logger),
		ModifyResponse: modifyResponse(),
		ErrorHandler:   errorHandler(logger),
	}
	return &Proxy{
		server: &http.Server{
			Addr:      fmt.Sprintf(":%d", port),
			Handler:   handler(logger, reverseProxy, conf, domain.NewMatcher(), mocksEnabled),
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
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
		for _, rule := range route.Rewrite {
			if matches := rule.PathPattern.MatchString(path.Clean(req.URL.Path)); matches {
				if err := rewrite(rule, req); err != nil {
					logger.Println(fmt.Sprintf("failed to rewrite request. %v", err))
					continue
				}
				break
			}
		}

		// set any proxy pass headers from config
		for name, value := range route.ProxyPassHeaders {
			req.Header.Set(name, value)
		}
	}
}

func modifyResponse() func(*http.Response) error {
	return func(res *http.Response) error {
		route, ok := res.Request.Context().Value(routeCtxKey).(*domain.Route)
		if !ok {
			// if route not set, then default backend was used and no route match config available
			return nil
		}

		for k, v := range route.ProxyResponseHeaders {
			res.Header.Set(k, v)
		}

		return nil
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
		case domain.RouteTypeRedirect:
			to := replaceURL(matchedRoute.PathPattern, matchedRoute.Redirect.To, r.URL)
			u, err := url.Parse(to)
			if err != nil {
				logger.Printf(err.Error())
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte("Bad gateway"))
				return
			}

			http.Redirect(w, r, u.String(), redirectStatusCode(matchedRoute.Redirect.Type))
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
		case domain.RouteTypeRedirect:
			if route.Redirect == nil {
				return nil, errors.New("missing redirect in config")
			}
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

func rewrite(rule domain.Rewrite, req *http.Request) error {
	to := path.Clean(replaceURL(rule.PathPattern, rule.To, req.URL))
	u, e := url.Parse(to)
	if e != nil {
		return fmt.Errorf("rewritten URL is not valid. %w", e)
	}

	req.URL.Path = u.Path
	req.URL.RawPath = u.RawPath
	if u.RawQuery != "" {
		req.URL.RawQuery = u.RawQuery
	}

	return nil
}

func replaceURL(pattern *domain.PathPattern, to string, u *url.URL) string {
	uri := u.RequestURI()
	match := pattern.FindStringSubmatchIndex(uri)
	result := pattern.ExpandString([]byte(""), to, uri, match)
	return string(result[:])
}

func redirectStatusCode(method string) int {
	if method == "permanent" || method == "" {
		return http.StatusMovedPermanently
	}
	return http.StatusFound
}
