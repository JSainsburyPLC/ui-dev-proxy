package domain

import (
	"encoding/json"
	"net/url"
	"regexp"
)

const (
	RouteTypeProxy    = "proxy"
	RouteTypeMock     = "mock"
	RouteTypeRedirect = "redirect"
)

type Config struct {
	Routes []Route `json:"routes"`
}

type Route struct {
	Type                 string            `json:"type"`
	PathPattern          *PathPattern      `json:"path_pattern"`
	Backend              *Backend          `json:"backend"`
	Mock                 *Mock             `json:"mock"`
	Rewrite              []Rewrite         `json:"rewrite"`
	Redirect             *Redirect         `json:"redirect"`
	ProxyPassHeaders     map[string]string `json:"proxy_pass_headers"`
	ProxyResponseHeaders map[string]string `json:"proxy_response_headers"`
}

type Rewrite struct {
	PathPattern *PathPattern `json:"path_pattern"`
	To          string       `json:"to"`
}

type Redirect struct {
	To   string `json:"to"`
	Type string `json:"type"` // either permanent or temporary. Defaults to permanent if not provided
}

type PathPattern struct {
	*regexp.Regexp
}

func (p *PathPattern) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	r, err := regexp.Compile(s)
	if err != nil {
		return err
	}

	p.Regexp = r

	return nil
}

type Backend struct {
	*url.URL
}

func (p *Backend) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	u, err := url.Parse(s)
	if err != nil {
		return err
	}

	p.URL = u

	return nil
}

// TODO: the path arg here is a leaky abstraction - fix it
type ConfigProvider func(path string) (Config, error)
