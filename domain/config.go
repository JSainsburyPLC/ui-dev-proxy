package domain

import (
	"encoding/json"
	"net/url"
	"regexp"
)

const (
	RouteTypeProxy = "proxy"
	RouteTypeMock  = "mock"
)

type Config struct {
	Routes []Route `json:"routes"`
}

type Route struct {
	Type        string            `json:"type"`
	PathPattern *PathPattern      `json:"path_pattern"`
	Backend     *Backend          `json:"backend"`
	Mock        *Mock             `json:"mock"`
	Rewrite     map[string]string `json:"rewrite"`
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
