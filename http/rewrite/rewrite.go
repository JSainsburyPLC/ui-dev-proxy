package rewrite

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
)

type Rule struct {
	pattern string
	to      string
	regexp  *regexp.Regexp
}

func NewRule(pattern, to string) (Rule, error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return Rule{}, err
	}

	return Rule{
		pattern: pattern,
		to:      to,
		regexp:  reg,
	}, nil
}

func (r *Rule) Rewrite(req *http.Request) (bool, error) {
	oriPath := req.URL.Path

	if !r.regexp.MatchString(oriPath) {
		return false, nil
	}

	to := path.Clean(r.Replace(req.URL))
	u, e := url.Parse(to)
	if e != nil {
		return false, fmt.Errorf("rewritten URL is not valid. %w", e)
	}

	req.URL.Path = u.Path
	req.URL.RawPath = u.RawPath
	if u.RawQuery != "" {
		req.URL.RawQuery = u.RawQuery
	}

	return true, nil
}

func (r *Rule) Replace(u *url.URL) string {
	uri := u.RequestURI()
	patternRegexp := regexp.MustCompile(r.pattern)
	match := patternRegexp.FindStringSubmatchIndex(uri)
	result := patternRegexp.ExpandString([]byte(""), r.to, uri, match)
	return string(result[:])
}
