package rewrite_test

import (
	"net/http"
	"testing"

	"github.com/JSainsburyPLC/ui-dev-proxy/http/rewrite"
	"github.com/stretchr/testify/assert"
)

func TestRewrite(t *testing.T) {
	tests := map[string]struct {
		pattern string
		to      string
		before  string
		after   string
		matched bool
	}{
		"constant": {
			pattern: "/a",
			to:      "/b",
			before:  "/a",
			after:   "/b",
			matched: true,
		},
		"preserves original URL if no match": {
			pattern: "/a",
			to:      "/b",
			before:  "/c",
			after:   "/c",
			matched: false,
		},
		"match group": {
			pattern: "/api/(.*)",
			to:      "/$1",
			before:  "/api/my-endpoint",
			after:   "/my-endpoint",
			matched: true,
		},
		"multiple match groups": {
			pattern: "/a/(.*)/b/(.*)",
			to:      "/x/y/$1/z/$2",
			before:  "/a/oo/b/qq",
			after:   "/x/y/oo/z/qq",
			matched: true,
		},
		"encoded characters": {
			pattern: "/a/(.*)",
			to:      "/b/$1",
			before:  "/a/x-1%2F",
			after:   "/b/x-1%2F",
			matched: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest("GET", test.before, nil)
			if err != nil {
				t.Fatalf("failed to create request %v %v", test, err)
			}
			rule, err := rewrite.NewRule(test.pattern, test.to)
			if err != nil {
				t.Fatal(err)
			}

			matched, err := rule.Rewrite(req)

			assert.NoError(t, err)
			assert.Equal(t, test.after, req.URL.EscapedPath())
			assert.Equal(t, test.matched, matched)
		})
	}
}
