package domain

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatcher_Matches(t *testing.T) {
	tests := map[string]struct {
		mock             Mock
		request          *http.Request
		expectedResponse Response
		expectedOK       bool
	}{
		"matches": {
			mock:             mockUser,
			request:          httptest.NewRequest(http.MethodGet, "/user?name=Peter&age=30", nil),
			expectedResponse: Response{Body: `{"name":"Peter Ndlovu"}`, Status: 200},
			expectedOK:       true,
		},
		"matches with regex": {
			mock:             mockPet,
			request:          httptest.NewRequest(http.MethodPost, "/pet/search", strings.NewReader(`{"name": "Dave"}`)),
			expectedResponse: Response{Body: `{"error":"No pet called 'Dave'"}`, Status: 400, Headers: map[string]string{"X-Correlation-ID": "1234567890"}},
			expectedOK:       true,
		},
		"no match if path different": {
			mock:             mockUser,
			request:          httptest.NewRequest(http.MethodGet, "/something?name=Peter&age=30", nil),
			expectedResponse: Response{},
			expectedOK:       false,
		},
		"no match if body different": {
			mock:             mockPet,
			request:          httptest.NewRequest(http.MethodPost, "/pet/search", strings.NewReader(`{"name": "John"}`)),
			expectedResponse: Response{},
			expectedOK:       false,
		},
		"with query params - page 1": {
			mock: Mock{
				MatchRequest: MatchRequest{
					Method: "GET",
					Path:   "/user",
					Query:  "page=1",
				},
				Response: Response{
					Status: 200,
					Body:   `{"field": "page1"}`,
				},
			},
			request:          httptest.NewRequest(http.MethodGet, "/user?page=1", nil),
			expectedResponse: Response{Body: `{"field": "page1"}`, Status: 200},
			expectedOK:       true,
		},
		"with query params - page 2": {
			mock: Mock{
				MatchRequest: MatchRequest{
					Method: "GET",
					Path:   "/user",
					Query:  "page=2",
				},
				Response: Response{
					Status: 200,
					Body:   `{"field": "page1"}`,
				},
			},
			request:          httptest.NewRequest(http.MethodGet, "/user?page=2", nil),
			expectedResponse: Response{Body: `{"field": "page2"}`, Status: 200},
			expectedOK:       true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matcher := NewMatcher()
			match := matcher.Match(test.request, test.mock)
			assert.Equal(t, test.expectedOK, match)
		})
	}
}

var mockUser = Mock{
	MatchRequest: MatchRequest{
		Method: "GET",
		Path:   "/user",
		Query:  `name=Peter&age=30`,
	},
	Response: Response{
		Status: 200,
		Body:   `{"name":"Peter Ndlovu"}`,
	},
}

var mockPet = Mock{
	MatchRequest: MatchRequest{
		Method: "POST",
		Path:   "/pet/s.*h",
		Body:   `{"name": "Dave"}`,
	},
	Response: Response{
		Status: 400,
		Body:   `{"error":"No pet called 'Dave'"}`,
		Headers: map[string]string{
			"X-Correlation-ID": "1234567890",
		},
	},
}
