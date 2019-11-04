# UI Dev Proxy

[![Build Status](https://img.shields.io/travis/JSainsburyPLC/ui-dev-proxy.svg?logo=travis&style=for-the-badge)](https://travis-ci.org/JSainsburyPLC/ui-dev-proxy)

Proxy tool for development of UIs.

## Installation

```
go get -u github.com/JSainsburyPLC/ui-dev-proxy
```

## Usage

```
# start proxy server with default backend and config file
ui-dev-proxy start -u https://default-backend-url.example.com -c proxy-config.json
```

For additional options see help

```
ui-dev-proxy start --help
```

## How it works

The proxy can handle requests in 3 different ways:

* Pass through to backend when path matches pattern in config route
* Return a mock response when request matches config route (when mocks are enabled)
* Pass through to default backend, if request doesn't match any config routes

## Configuring your UI app

Routes are configured in a JSON file, which is passed to the proxy using the "-c" flag.

See `examples/config.json`

### Proxy type routes

```
{
  "type": "proxy", // Required
  "path_pattern": "^/test-ui/.*", // regex to match request path. Required
  "backend": "http://localhost:3000", // backend scheme and host to proxy to. Required
  "rewrite": { // optional rewrite rules
    "/test-ui/(.*)": "/$1"
  }
}
```

### Mock type routes

```
{
  "type": "mock", // Required
  "mock": {
    "request": { // parameters to match the inbound request on.
      "method": "GET", // match the method of the request. Optional
      "path": "^/api/v1/product/.*", // match the path of the request. Required
      "query": "include=.*" // match the query string of the request. Optional
    },
    "response": { // definition of the mock data to respond with.
      "status": 200, // the status code. Required
      "body": "mocks/product.json", // string body, or path to JSON file. Required
      "cookies": [ // set cookies. Optional
        {
          "name": "SOME_COOKIE",
          "value": "1234567890",
          "maxAge": 604800
        }
      ]
    }
  }
}
```

## Development

### Release

Releases are managed using goreleaser. Download a binary from a release [here](https://github.com/goreleaser/goreleaser/releases) and ensure it is on your PATH: 

To release:
-	Tag the commit you want to release, e.g. `git tag v0.1.1`
-	run `make release` 

You will need a github personal access token bound to env var `GITHUB_TOKEN`

You can dry run the release by running `make test-release`
