// Package airtable provides a high-level client to the Airtable API
// that allows the consumer to drop to a low-level request client when
// needed.
package airtable

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	defaultRootURL = "https://api.airtable.com"
	defaultVersion = "v0"
)

func makeURL(path string) {

}

// Client represents an interface to communicate with the Airtable API
type Client struct {
	APIKey  string
	BaseID  string
	Version string
	RootURL string
}

// ErrClientSetupError is returned when the client is missing APIKey
// or BaseID
type ErrClientSetupError struct {
	msg string
}

func (e ErrClientSetupError) Error() string {
	return e.msg
}

// ErrClientRequestError is returned when the client runs into
// problems with a request
type ErrClientRequestError struct {
	msg string
}

func (e ErrClientRequestError) Error() string {
	return e.msg
}

func (c *Client) checkSetup() error {
	if c.BaseID == "" {
		return ErrClientSetupError{"Client missing BaseID"}
	}
	if c.APIKey == "" {
		return ErrClientSetupError{"Client missing APIKey"}
	}
	if c.Version == "" {
		c.Version = defaultVersion
	}
	if c.RootURL == "" {
		c.RootURL = defaultRootURL
	}
	return nil
}

func (c *Client) makeURL(resource string, options QueryEncoder) string {
	q := options.Encode()
	url := fmt.Sprintf("%s/%s/%s/%s?%s",
		c.RootURL, c.Version, c.BaseID, resource, q)
	return url
}

// QueryEncoder encodes options to a query string
type QueryEncoder interface {
	Encode() string
}

type errorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
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

// RequestBytes makes a raw request to the Airtable API
func (c *Client) RequestBytes(resource string, options QueryEncoder) ([]byte, error) {
	var err error

	if err = c.checkSetup(); err != nil {
		return nil, err
	}

	if options == nil {
		options = url.Values{}
	}

	url := c.makeURL(resource, options)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}
	h := make(http.Header)
	h.Add("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	req.Header = h

	var httpclient http.Client
	resp, err := httpclient.Do(req)
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
