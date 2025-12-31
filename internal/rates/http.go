package rates

import (
	"crypto/tls"
	"net/http"
	"time"
)

// NewHTTPClient creates an HTTP client with optional TLS configuration.
// Set skipTLSVerify to true for servers with misconfigured certificate chains
// (e.g., servers that don't send intermediate certificates).
func NewHTTPClient(timeout time.Duration, skipTLSVerify bool) *http.Client {
	transport := &http.Transport{}

	if skipTLSVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// DefaultHTTPClient returns a standard HTTP client with 30s timeout.
func DefaultHTTPClient() *http.Client {
	return NewHTTPClient(30*time.Second, false)
}

// InsecureHTTPClient returns an HTTP client that skips TLS verification.
// Use this for servers with broken certificate chains (missing intermediate certs).
// WARNING: This disables certificate verification. Only use for known servers.
func InsecureHTTPClient() *http.Client {
	return NewHTTPClient(30*time.Second, true)
}
