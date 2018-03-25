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
	"os"
	"reflect"

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
	var err error

	// panic if the client isn't setup correctly to make a request
	c.checkSetup()

	if options == nil {
		options = url.Values{}
	}

	url := c.makeURL(endpoint, options)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header = make(http.Header)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

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
		panic("airtable: missing HTTP client")
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

func handleString(key string, f *reflect.Value, v *interface{}) {
	str, ok := (*v).(string)
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as string", key))
	}
	f.SetString(str)
}
func handleInt(key string, f *reflect.Value, v *interface{}) {
	// JavaScript/JSON doesn't have ints, only float64s
	n, ok := (*v).(float64)
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as int", key))
	}
	f.SetInt(int64(n))
}
func handleFloat(key string, f *reflect.Value, v *interface{}) {
	// JavaScript/JSON doesn't have ints, only float64s
	n, ok := (*v).(float64)
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as int", key))
	}
	f.SetFloat(n)
}
func handleSlice(key string, f *reflect.Value, v *interface{}) {
	s, ok := (*v).([]interface{})
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as slice", key))
	}

	dst := reflect.MakeSlice(f.Type(), len(s), cap(s))

	for i, v := range s {
		elem := dst.Index(i)
		switch elem.Kind() {
		case reflect.Struct:
			handleStruct(key, &elem, &v)
		case reflect.Bool:
			handleBool(key, &elem, &v)
		case reflect.Int:
			handleInt(key, &elem, &v)
		case reflect.Float64:
			handleFloat(key, &elem, &v)
		case reflect.String:
			handleString(key, &elem, &v)
		case reflect.Slice:
			handleSlice(key, &elem, &v)
		default:
			panic(fmt.Sprintf("UNHANDLED CASE: %s of kind %s", key, elem.Kind()))
		}

	}
	f.Set(dst)
}
func handleStruct(key string, s *reflect.Value, v *interface{}) {
	maybeParse := s.Addr().MethodByName("SelfParse")
	if maybeParse.Kind() == reflect.Func {
		args := []reflect.Value{reflect.ValueOf(v)}
		maybeParse.Call(args)
		return
	}

	m, ok := (*v).(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as struct", key))
	}

	sType := s.Type()
	for i := 0; i < sType.NumField(); i++ {
		f := s.Field(i)
		fType := sType.Field(i)
		key := fType.Name
		if from, ok := fType.Tag.Lookup("json"); ok {
			key = from
		}

		fmt.Println("key", key)
		fmt.Println("m", m)

		v := m[key]
		switch f.Kind() {
		case reflect.Struct:
			handleStruct(key, &f, &v)
		case reflect.Bool:
			handleBool(key, &f, &v)
		case reflect.Int:
			handleInt(key, &f, &v)
		case reflect.Float64:
			handleFloat(key, &f, &v)
		case reflect.String:
			handleString(key, &f, &v)
		case reflect.Slice:
			handleSlice(key, &f, &v)
		default:
			panic(fmt.Sprintf("UNHANDLED CASE: %s of kind %s", key, f.Kind()))
		}
	}
}
func handleBool(key string, f *reflect.Value, v *interface{}) {
	b, ok := (*v).(bool)
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as bool", key))
	}
	f.SetBool(b)
}
func handleInterface(key string, f *reflect.Value, v *interface{}) {
	f.Set(reflect.ValueOf(*v))
}
