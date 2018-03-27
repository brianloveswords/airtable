// Package airtable provides a high-level client to the Airtable API
// that allows the consumer to drop to a low-level request client when
// needed.
package airtable

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"go.uber.org/ratelimit"
)

var limiter = ratelimit.New(5) // per second

const (
	defaultRootURL = "https://api.airtable.com"
	defaultVersion = "v0"
)

// Client represents an interface to communicate with the Airtable API
type Client struct {
	APIKey     string
	BaseID     string
	Version    string
	RootURL    string
	HTTPClient *http.Client
}

// RequestBytes makes a raw request to the Airtable API
func (c *Client) RequestBytes(method string, endpoint string, options QueryEncoder) ([]byte, error) {
	return c.RequestWithBody(method, endpoint, options, http.NoBody)
}

// RequestWithBody makes a raw request to the Airtable API
func (c *Client) RequestWithBody(method string, endpoint string, options QueryEncoder, body io.Reader) ([]byte, error) {
	var err error

	// panic if the client isn't setup correctly to make a request
	c.checkSetup()

	if options == nil {
		options = url.Values{}
	}

	url := c.makeURL(endpoint, options)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = http.Header{}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Add("Content-Type", "application/json")

	if os.Getenv("AIRTABLE_NO_LIMIT") == "" {
		limiter.Take()
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = checkErrorResponse(bytes); err != nil {
		return bytes, err
	}

	return bytes, nil
}

func (c *Client) checkSetup() {
	if c.BaseID == "" {
		panic("airtable: Client missing BaseID")
	}
	if c.APIKey == "" {
		panic("airtable: Client missing APIKey")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Version == "" {
		c.Version = defaultVersion
	}
	if c.RootURL == "" {
		c.RootURL = defaultRootURL
	}
}

func (c *Client) makeURL(resource string, options QueryEncoder) string {
	q := options.Encode()
	url := fmt.Sprintf("%s/%s/%s/%s?%s",
		c.RootURL, c.Version, c.BaseID, resource, q)
	return url
}

// ErrClientRequestError is returned when the client runs into
// problems with a request
type ErrClientRequestError struct {
	msg string
}

func (e ErrClientRequestError) Error() string {
	return e.msg
}

type errorResponse struct {
	Error struct {
		Type    string
		Message string
	}
}

func checkErrorResponse(b []byte) error {
	var reqerr errorResponse
	if jsonerr := json.Unmarshal(b, &reqerr); jsonerr != nil {
		return jsonerr
	}
	if reqerr.Error.Type != "" {
		return ErrClientRequestError{reqerr.Error.Message}
	}
	return nil
}

// QueryEncoder encodes options to a query string
type QueryEncoder interface {
	Encode() string
}
