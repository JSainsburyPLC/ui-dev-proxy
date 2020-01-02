package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

// debugTransport wraps the http transport and logs the raw request and response
type debugTransport struct {
	originalTransport http.RoundTripper
}

func (c *debugTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	reqDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\n%v\n\n", string(reqDump))

	resp, err := c.originalTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	resDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\n%v\n\n", string(resDump))

	return resp, nil
}
