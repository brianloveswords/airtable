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

// Request makes a raw request to the Airtable API
func (c *Client) Request(method string, endpoint string, options QueryEncoder) ([]byte, error) {
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
		return bytes, ErrClientRequestError{
			err:    err,
			url:    url,
			method: method,
		}
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
	p := resource
	uri := fmt.Sprintf("%s/%s/%s/%s?%s",
		c.RootURL, c.Version, c.BaseID, p, q)
	return uri
}

// ErrClientRequestError is returned when the client runs into
// problems with a request
type ErrClientRequestError struct {
	err    error
	method string
	url    string
}

func (e ErrClientRequestError) Error() string {
	return fmt.Sprintf("client request error: %s %s: %s", e.method, e.url, e.err)
}

type genericErrorResponse struct {
	Error interface{} `json:"error"`
}

func checkErrorResponse(b []byte) error {
	var generic genericErrorResponse
	if err := json.Unmarshal(b, &generic); err != nil {
		return fmt.Errorf("couldn't unmarshal response: %s", err)
	}
	if generic.Error == nil {
		return nil
	}
	return fmt.Errorf("%s", generic.Error)
}

// QueryEncoder encodes options to a query string
type QueryEncoder interface {
	Encode() string
}
