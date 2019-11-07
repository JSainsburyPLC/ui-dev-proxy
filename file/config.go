package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/JSainsburyPLC/ui-dev-proxy/domain"
)

func ConfigProvider() domain.ConfigProvider {
	return func(path string) (domain.Config, error) {
		f, err := os.Open(path)
		if err != nil {
			return domain.Config{}, err
		}

		var c domain.Config
		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return domain.Config{}, err
		}

		configDir := filepath.Dir(f.Name())
		if configDir != "/" {
			configDir = configDir + "/"
		}

		for _, r := range c.Routes {
			if r.Type != domain.RouteTypeMock {
				if r.Redirect != nil {
					redirectType := r.Redirect.Type
					if redirectType != "permanent" && redirectType != "temporary" {
						return domain.Config{}, fmt.Errorf("invalid redirect type '%s'", redirectType)
					}
				}

				continue
			}

			if r.Mock == nil {
				return domain.Config{}, errors.New("missing mock config on mock type route")
			}

			r.Mock.MatchRequest.Body, err = getBody(r.Mock.MatchRequest.Body, configDir)
			if err != nil {
				return domain.Config{}, err
			}

			r.Mock.Response.Body, err = getBody(r.Mock.Response.Body, configDir)
			if err != nil {
				return domain.Config{}, err
			}
		}

		return c, nil
	}
}

func getBody(body string, configDir string) (string, error) {
	if !strings.HasSuffix(body, ".json") {
		return body, nil
	}

	f, err := os.Open(configDir + body)
	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
