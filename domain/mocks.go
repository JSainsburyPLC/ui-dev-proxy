package domain

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
)

// Mock represents an individual mock
type Mock struct {
	MatchRequest MatchRequest `json:"request"`
	Response     Response     `json:"response"`
}

// MatchRequest is the user defined matcher that we check incoming requests against.
// A mock is considered to match if MatchRequest is equal to the incoming request
type MatchRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query"`
	Body   string `json:"body"`
}

// Response is returned to the consumer if the MockRequest matches. If multiple requests match
// the first Response is returned
type Response struct {
	Status  int               `json:"status"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Cookies []Cookie          `json:"cookies"`
}

// Cookie is added to a `Set-Cookie` header in the mock response
type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	MaxAge int    `json:"maxAge"`
}

// Matcher is the core service that orchestrates comparing the incoming request against the matchers
// and returning the response if the request is matched
type Matcher struct {
	matchers []matcher
}

// NewMatcher creates a Matcher composed of all of the registered matchers
func NewMatcher() Matcher {
	return Matcher{
		matchers: []matcher{matchesMethod, matchesPath, matchesQuery, matchesBody},
	}
}

// Match matches a mock against all matchers
func (m Matcher) Match(r *http.Request, mock Mock) bool {
	found := true
	for _, matcher := range m.matchers {
		if ok := matcher(r, mock); !ok {
			found = false
			break
		}
	}
	if found == true {
		return found
	}
	return false
}

type matcher func(r *http.Request, mock Mock) bool

var matchesPath matcher = func(r *http.Request, mock Mock) bool {
	receivedPath := r.URL.Path
	mockPath := mock.MatchRequest.Path
	if receivedPath == mockPath {
		return true
	}
	matched, err := regexp.MatchString(mockPath, receivedPath)
	if matched && err == nil {
		return true
	}
	return false
}

var matchesQuery matcher = func(r *http.Request, mock Mock) bool {
	if mock.MatchRequest.Query == "" {
		return true
	}

	if mock.MatchRequest.Query == r.URL.RawQuery {
		return true
	}

	receivedQuery := r.URL.Query()
	mockQuery, err := url.ParseQuery(mock.MatchRequest.Query)
	if err != nil {
		return false
	}

	for key, values := range mockQuery {
		var err error
		var match bool

		for _, field := range receivedQuery[key] {
			for _, value := range values {
				match, err = regexp.MatchString(value, field)
				if err != nil {
					return false
				}
			}

			if match {
				break
			}
		}

		if !match {
			return false
		}
	}

	return true
}

var matchesMethod matcher = func(r *http.Request, mock Mock) bool {
	if r.Method == mock.MatchRequest.Method {
		return true
	}
	if mock.MatchRequest.Method == "" {
		return true
	}
	return false
}

var matchesBody = func(req *http.Request, mock Mock) bool {
	mockBody := mock.MatchRequest.Body

	if len(mockBody) == 0 {
		return true
	}

	if req.Body == nil {
		return false
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return false
	}
	if len(body) == 0 {
		return false
	}

	// replace body so it can be read again
	req.Body = ioutil.NopCloser(bytes.NewReader(body))

	// Perform exact string match
	bodyStr := string(body)
	if bodyStr == mockBody {
		return true
	}

	// Perform regexp match
	match, _ := regexp.MatchString(mockBody, bodyStr)
	if match {
		return true
	}

	// Perform JSON match but converting request and mock to a map before comparing
	var reqJSON map[string]interface{}
	reqJSONErr := json.Unmarshal(body, &reqJSON)

	var matchJSON map[string]interface{}
	specJSONErr := json.Unmarshal([]byte(mockBody), &matchJSON)

	isJSON := reqJSONErr == nil && specJSONErr == nil
	if isJSON && reflect.DeepEqual(reqJSON, matchJSON) {
		return true
	}

	return false
}
