// Package airtable provides a high-level client to the Airtable API
// that allows the consumer to drop to a low-level request client when
// needed.
package airtable

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"go.uber.org/ratelimit"
)

var limiter = ratelimit.New(5) // per second

const (
	defaultRootURL = "https://api.airtable.com"
	defaultVersion = "v0"
)

// Client represents an interface to communicate with the Airtable API.
//
// - APIKey: api key to use for each request. Requests will panic
// if this is not set.
//
// - BaseID: base this client will operate against. Requests will panic
// if this not set.
//
// - Version: version of the API to use. Defaults to "v0".
//
// - RootURL: root URL to use. defaults to "https://api.airtable.com"
//
// - HTTPClient: http.Client instance to use. Defaults to
// http.DefaultClient
type Client struct {
	APIKey     string
	BaseID     string
	Version    string
	RootURL    string
	HTTPClient *http.Client
}

// Request makes an HTTP request to the Airtable API without a body. See
// RequestWithBody for documentation.
func (c *Client) Request(
	method string,
	endpoint string,
	options QueryEncoder,
) ([]byte, error) {
	return c.RequestWithBody(method, endpoint, options, http.NoBody)
}

// RequestWithBody makes an HTTP request to the Airtable API. endpoint
// will be combined with the client's RootlURL, Version and BaseID, to
// create the complete URL. endpoint is expected to already be encoded;
// if necessary, use url.PathEscape before passing RequestWithBody.
//
// If client is missing APIKey or BaseID, this method will panic.
func (c *Client) RequestWithBody(
	method string,
	endpoint string,
	options QueryEncoder,
	body io.Reader,
) ([]byte, error) {
	var err error

	// will panic if the client isn't setup correctly to make a request
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
			Err:    err,
			URL:    url,
			Method: method,
		}
	}

	return bytes, nil
}

// Table returns a new Table
func (c *Client) Table(name string) Table {
	return Table{
		client: c,
		name:   name,
	}
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
// problems making a request.
type ErrClientRequestError struct {
	Err    error
	Method string
	URL    string
}

func (e ErrClientRequestError) Error() string {
	return fmt.Sprintf("client request error: %s %s: %s", e.Method, e.URL, e.Err)
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

// Record is a convenience struct for anonymous inclusion in
// user-constructed record structs.
type Record struct {
	ID          string
	CreatedTime time.Time
}

// Fields is used in NewRecord for constructing new records.
type Fields map[string]interface{}

// NewRecord is a convenience method for applying a map of fields to a
// record container when the Fields struct is anonymous.
func NewRecord(container interface{}, data Fields) {
	// iterating over the container fields and applying those keys to
	// the passed in fields would be "safer", but it could possibly
	// mask user error if data is the complete wrong fit. instead we
	// can iterate over data and apply that to the container, and fail
	// early if there isn't a matching field.
	ref := reflect.ValueOf(container).Elem()
	typ := ref.Type()
	fields := ref.FieldByName("Fields")
	for k, v := range data {
		f := fields.FieldByName(k)
		val := reflect.ValueOf(v)
		if !f.IsValid() {
			errstr := fmt.Sprintf("cannot find field %s.%s", typ, k)
			panic(errstr)
		}
		if fkind, vkind := f.Kind(), val.Kind(); fkind != vkind {
			errstr := fmt.Sprintf("type error setting %s.%s: %s != %s", typ, k, fkind, vkind)
			panic(errstr)
		}
		f.Set(val)
	}
}

type deleteResponse struct {
	Deleted bool
	ID      string
}

// Table represents an table in a base and provides methods for
// interacting with records in the table.
type Table struct {
	name   string
	client *Client
}

// Get looks up a record from the table by ID and stores in in the
// object pointed to by recordPtr.
func (t *Table) Get(id string, recordPtr interface{}) error {
	bytes, err := t.client.Request("GET", t.makePath(id), nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, recordPtr)
}

// Update sends the updated record pointed to by recordPtr to the table
func (t *Table) Update(recordPtr interface{}) error {
	id, err := getID(recordPtr)
	if err != nil {
		return err
	}
	body, err := getJSONBody(recordPtr)
	if err != nil {
		return err
	}
	_, err = t.client.RequestWithBody("PATCH", t.makePath(id), Options{}, body)
	if err != nil {
		return err
	}
	return nil
}

// Create makes a new record in the table using the record pointed to by
// recordPtr. On success, updates the ID and CreatedTime of the object
// pointed to by recordPtr.
func (t *Table) Create(recordPtr interface{}) error {
	body, err := getJSONBody(recordPtr)
	if err != nil {
		return err
	}
	res, err := t.client.RequestWithBody("POST", t.makePath(""), Options{}, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(res, recordPtr)
}

// Delete removes a record from the table. On success, ID and
// CreatedTime of the object pointed to by recordPtr are removed.
func (t *Table) Delete(recordPtr interface{}) error {
	id, err := getID(recordPtr)
	if err != nil {
		return err
	}
	res, err := t.client.Request("DELETE", t.makePath(id), Options{})
	if err != nil {
		return err
	}
	deleted := deleteResponse{}
	if err := json.Unmarshal(res, &deleted); err != nil {
		return err
	}
	if !deleted.Deleted {
		return fmt.Errorf("error: did not delete %s", res)
	}
	markAsDeleted(recordPtr)
	return nil
}

// List queries the table for list of records and stores it in the
// object pointed to by listPtr. By default, List will recurse to get
// all of the records until there are no more left to get, but this can
// be overriden by using the MaxRecords option. See Options for a
// complete list of the options that are supported.
func (t *Table) List(listPtr interface{}, options *Options) error {
	if options == nil {
		options = &Options{}
	}

	oneRecord := reflect.TypeOf(listPtr).Elem().Elem()
	options.typ = oneRecord

	bytes, err := t.client.Request("GET", t.makePath(""), options)
	if err != nil {
		return err
	}

	responseType := reflect.StructOf([]reflect.StructField{
		{Name: "Records", Type: reflect.TypeOf(listPtr).Elem()},
		{Name: "Offset", Type: reflect.TypeOf("")},
	})

	container := reflect.New(responseType)
	err = json.Unmarshal(bytes, container.Interface())
	if err != nil {
		return err
	}

	recordList := container.Elem().FieldByName("Records")
	list := reflect.ValueOf(listPtr).Elem()
	for i := 0; i < recordList.Len(); i++ {
		entry := recordList.Index(i)
		list = reflect.Append(list, entry)
	}
	reflect.ValueOf(listPtr).Elem().Set(list)

	offset := container.Elem().FieldByName("Offset").String()
	if offset != "" {
		options.offset = offset
		return t.List(listPtr, options)
	}
	return nil
}

func (t *Table) makePath(id string) string {
	n := url.PathEscape(t.name)
	if id == "" {
		return n
	}
	return path.Join(n, id)
}

func markAsDeleted(recordPtr interface{}) {
	emptyTime := reflect.ValueOf(time.Time{})
	reflect.ValueOf(recordPtr).Elem().FieldByName("ID").SetString("")
	reflect.ValueOf(recordPtr).Elem().FieldByName("CreatedTime").Set(emptyTime)
}

func getJSONBody(r interface{}) (io.Reader, error) {
	f, err := getFields(r)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	jsonstr := fmt.Sprintf(`{"fields": %s}`, b)
	body := strings.NewReader(jsonstr)
	return body, nil
}

func getFields(e interface{}) (interface{}, error) {
	fields := reflect.ValueOf(e).Elem().FieldByName("Fields")
	if !fields.IsValid() {
		return nil, errors.New("getFields: missing Fields")
	}
	if fields.Kind() != reflect.Struct {
		return nil, errors.New("getFields: Fields not a struct")
	}
	return fields.Interface(), nil
}

func getID(e interface{}) (string, error) {
	id := reflect.ValueOf(e).Elem().FieldByName("ID")
	if !id.IsValid() {
		return "", errors.New("getID: missing ID")
	}
	if id.Kind() != reflect.String {
		return "", errors.New("getID: ID not a string")
	}
	return id.String(), nil
}

func debugLog(s string, a ...interface{}) {
	fmt.Printf("DEBUG: "+s+"\n", a...)
}

// QueryEncoder encodes options to a query string.
type QueryEncoder interface {
	Encode() string
}
