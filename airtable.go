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
	"path"
	"reflect"
	"strings"
	"time"

	"go.uber.org/ratelimit"
)

var (
	DefaultRootURL    = "https://api.airtable.com"
	DefaultVersion    = "v0"
	DefaultHTTPClient = http.DefaultClient
	DefaultLimiter    = RateLimiter(5) // per second
)

// RateLimiter makes a new rate limiter using n as the number of
// requests per second that is allowed. If 0 is passed, the limiter will
// be unlimited.
func RateLimiter(n int) ratelimit.Limiter {
	if n == 0 {
		return ratelimit.NewUnlimited()
	}
	return ratelimit.New(n)
}

// QueryEncoder encodes options to a query string.
type QueryEncoder interface {
	Encode() string
}

// Client represents an interface to communicate with the Airtable API.
//
// - APIKey: api key to use for each request. Requests will panic
// if this is not set.
//
// - BaseID: base this client will operate against. Requests will panic
// if this not set.
//
// - Version: version of the API to use.
//
// - RootURL: root URL to use.
//
// - HTTPClient: http.Client instance to use.
// http.DefaultClient
//
// - Limit: max requests to make per second.
type Client struct {
	APIKey     string
	BaseID     string
	Version    string
	RootURL    string
	HTTPClient *http.Client
	Limiter    ratelimit.Limiter
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

// ErrClientRequest is returned when the client runs into
// problems making a request.
type ErrClientRequest struct {
	Err    error
	Method string
	URL    string
}

func (e ErrClientRequest) Error() string {
	return fmt.Sprintf("airtable client request error: %s %s: %s", e.Method, e.URL, e.Err)
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

	// finish setup or panic if the client isn't configured correctly
	c.checkSetup()

	if options == nil {
		options = url.Values{}
	}
	url := c.makeURL(endpoint, options)
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, ErrClientRequest{
			Err:    err,
			URL:    url,
			Method: method,
		}
	}

	c.makeHeader(req)

	// adhere to the rate limit
	c.Limiter.Take()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, ErrClientRequest{
			Err:    err,
			URL:    url,
			Method: method,
		}
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrClientRequest{
			Err:    err,
			URL:    url,
			Method: method,
		}
	}

	if err = checkErrorResponse(bytes); err != nil {
		return bytes, ErrClientRequest{
			Err:    err,
			URL:    url,
			Method: method,
		}
	}

	return bytes, nil
}

// Table returns a new Table that will use this client and operate
// against the table with the passed in name
func (c *Client) Table(name string) Table {
	return Table{
		client: c,
		name:   name,
	}
}

func (c *Client) getLimiter() ratelimit.Limiter {
	return c.Limiter
}

func (c *Client) makeHeader(r *http.Request) {
	r.Header = http.Header{}
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	r.Header.Add("Content-Type", "application/json")
}

func (c *Client) checkSetup() {
	if c.BaseID == "" {
		panic("airtable: Client missing BaseID")
	}
	if c.APIKey == "" {
		panic("airtable: Client missing APIKey")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = DefaultHTTPClient
	}
	if c.Version == "" {
		c.Version = DefaultVersion
	}
	if c.RootURL == "" {
		c.RootURL = DefaultRootURL
	}
	if c.Limiter == nil {
		c.Limiter = DefaultLimiter
	}
}

func (c *Client) makeURL(resource string, options QueryEncoder) string {
	q := options.Encode()
	p := resource
	uri := fmt.Sprintf("%s/%s/%s/%s?%s",
		c.RootURL, c.Version, c.BaseID, p, q)
	return uri
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
			errstr := fmt.Sprintf("airtable.NewRecord: cannot find field %s.%s", typ, k)
			panic(errstr)
		}
		if fkind, vkind := f.Kind(), val.Kind(); fkind != vkind {
			errstr := fmt.Sprintf("airtable.NewRecord: type error setting %s.%s: %s != %s", typ, k, fkind, vkind)
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

func validateRecordArg(recordPtr interface{}) {
	// must be:
	// ... a pointer
	typ := reflect.TypeOf(recordPtr)
	recordPtrKind := typ.Kind()
	if recordPtrKind != reflect.Ptr {
		panic(fmt.Errorf("airtable type error: recordPtr must be a pointer, got %s", recordPtrKind))
	}

	// ... to a struct
	record := typ.Elem()
	recordKind := record.Kind()
	if recordKind != reflect.Struct {
		panic(fmt.Errorf("airtable type error: recordPtr must point to a struct, got %s", recordKind))
	}

	// ... which has a field named "Fields" that's a struct
	fields, ok := record.FieldByName("Fields")
	if !ok {
		panic(fmt.Errorf("airtable type error: recordPtr must point to a struct with field 'Fields'"))
	}

	fieldsKind := fields.Type.Kind()
	if fieldsKind != reflect.Struct {
		panic(fmt.Errorf("airtable type error: recordPtr must point to a struct with field 'Fields' that is a struct, got %s", fieldsKind))
	}

	// ... and a field named "ID" that's a string
	id, ok := record.FieldByName("ID")
	if !ok {
		panic(fmt.Errorf("airtable type error: recordPtr must point to a struct with field 'ID'"))
	}

	idKind := id.Type.Kind()
	if idKind != reflect.String {
		panic(fmt.Errorf("airtable type error: recordPtr must point to a struct with field 'ID' that is a string, got %s", idKind))
	}
}

// Update sends the updated record pointed to by recordPtr to the table
func (t *Table) Update(recordPtr interface{}) error {
	// panic if the recordPtr doesn't point to a record.
	validateRecordArg(recordPtr)

	id := getID(recordPtr)

	// panic makeJSONBody errors because it's an upstream programming
	// error that needs to be fixed, not a user input error or a network
	// condition.
	body, err := makeJSONBody(recordPtr)
	if err != nil {
		panic(fmt.Errorf("airtable.Table#Update: unable to create JSON (%s)", err))
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
//
// recordPtr MUST have a Fields field that is a struct that can be
// marshaled to JSON or this method will panic.
func (t *Table) Create(recordPtr interface{}) error {
	// panic if the recordPtr doesn't point to a record.
	validateRecordArg(recordPtr)

	body, err := makeJSONBody(recordPtr)

	// panic if we can't create the JSON because it's an upstream
	// programming error that needs to be fixed, not a user input error
	// or a network condition.
	if err != nil {
		panic(fmt.Errorf("airtable.Table#Create: unable to create JSON (%s)", err))
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
	// panic if the recordPtr doesn't point to a record.
	validateRecordArg(recordPtr)

	id := getID(recordPtr)

	res, err := t.client.Request("DELETE", t.makePath(id), Options{})
	if err != nil {
		return fmt.Errorf("airtable.Table#Delete: request error %s", err)
	}
	deleted := deleteResponse{}
	if err := json.Unmarshal(res, &deleted); err != nil {
		return fmt.Errorf("airtable.Table#Delete: could not unpack request %s", err)
	}
	if !deleted.Deleted {
		return fmt.Errorf("airtable.Table#Delete: did not delete, %s", res)
	}
	markAsDeleted(recordPtr)
	return nil
}

// makeResponseContainer returns a new struct based on listPtr that can
// contain the response from a list query to an airtable. For example:
//
//  type BookRecord struct {
//		airtable.Record
//		Fields struct {
//			Title string
//			Author string
//		}
//  }
//  listPtr := &[]BookRecord{}
//
// Passing the above listPtr to makeResponseContainer will dynamically
// create a struct that looks like this:
//
//  struct {
//		Records []BookRecord
//		Offset  string
//  }
func makeResponseContainer(listPtr interface{}) reflect.Value {
	responseType := reflect.StructOf([]reflect.StructField{
		{Name: "Records", Type: reflect.TypeOf(listPtr).Elem()},
		{Name: "Offset", Type: reflect.TypeOf("string")},
	})
	return reflect.New(responseType)
}

func getOffset(listResponseContainer reflect.Value) string {
	return listResponseContainer.Elem().FieldByName("Offset").String()
}

func appendRecordsToList(listPtr interface{}, results reflect.Value) {
	var (
		original = reflect.ValueOf(listPtr).Elem()
		records  = results.Elem().FieldByName("Records")
		updated  = reflect.AppendSlice(original, records)
	)
	original.Set(updated)
}

// getRecordType will get the base element type from a pointer to a
// slice. For example: getRecordType(*[]string) -> string
func getRecordType(ps interface{}) reflect.Type {
	return reflect.TypeOf(ps).Elem().Elem()
}

func validateListArg(listPtr interface{}) {
	// must be:
	// ... a pointer
	typ := reflect.TypeOf(listPtr)
	listPtrKind := typ.Kind()
	if listPtrKind != reflect.Ptr {
		panic(fmt.Errorf("airtable type error: listPtr must be a pointer, got %s", listPtrKind))
	}

	// ... to a slice
	list := typ.Elem()
	listKind := list.Kind()
	if listKind != reflect.Slice {
		panic(fmt.Errorf("airtable type error: listPtr must point to a slice, got %s", listKind))
	}

	// ... whose elements are structs
	elem := list.Elem()
	elemKind := elem.Kind()
	if elemKind != reflect.Struct {
		panic(fmt.Errorf("airtable type error: listPtr must point to a slice of structs, got %s", elemKind))
	}

	// ... the structs have a field named "Fields" that's a struct
	fields, ok := elem.FieldByName("Fields")
	if !ok {
		panic(fmt.Errorf("airtable type error: listPtr must point to a slice of structs with field 'Fields'"))
	}

	fieldsKind := fields.Type.Kind()
	if fieldsKind != reflect.Struct {
		panic(fmt.Errorf("airtable type error: listPtr must point to a slice of structs with field 'Fields' that is a struct, got %s", fieldsKind))
	}

	// ... and a field named "ID" that's a string
	id, ok := elem.FieldByName("ID")
	if !ok {
		panic(fmt.Errorf("airtable type error: listPtr must point to a slice of structs with field 'ID'"))
	}

	idKind := id.Type.Kind()
	if idKind != reflect.String {
		panic(fmt.Errorf("airtable type error: listPtr must point to a slice of structs with field 'ID' that is a string, got %s", idKind))
	}
}

// List queries the table for list of records and stores it in the
// object pointed to by listPtr. By default, List will recurse to get
// all of the records until there are no more left to get, but this can
// be overriden by using the MaxRecords option. See Options for a
// complete list of the options that are supported.
//
// listPtr must be a pointer to a slice of records, which are structs
// that contain, at a minimum, `ID string` and `Fields struct {...}`
// fields. For example:
//
//  type BookRecord struct {
//		airtable.Record // provides ID and CreatedTime
//		Fields struct {
//			Title string
//			Author string
//		}
//  }
//  listPtr := &[]BookRecord{}
//
// This will be validated and cause a panic at runtime if listPtr is the
// wrong type.
func (t *Table) List(listPtr interface{}, options *Options) error {
	validateListArg(listPtr)

	if options == nil {
		options = &Options{}
	}

	// for "sort" and "fields" we need to have access to the type of
	// record so we can look up the JSON names of the fields.
	options.setType(getRecordType(listPtr))

	for {
		container := makeResponseContainer(listPtr)
		bytes, err := t.client.Request("GET", t.makePath(""), options)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, container.Interface())
		if err != nil {
			return err
		}
		appendRecordsToList(listPtr, container)
		options.offset = getOffset(container)
		if options.offset == "" {
			break
		}
	}
	return nil
}

func (t *Table) makePath(id string) string {
	name := url.PathEscape(t.name)
	if id == "" {
		return name
	}
	return path.Join(name, id)
}

func markAsDeleted(recordPtr interface{}) {
	var (
		record   = reflect.ValueOf(recordPtr).Elem()
		zeroTime = reflect.ValueOf(time.Time{})
		id       = record.FieldByName("ID")
		created  = record.FieldByName("CreatedTime")
	)
	if id.IsValid() {
		id.SetString("")
	}
	if created.IsValid() {
		created.Set(zeroTime)
	}
}

// makeJSONBody returns an io.Reader prepared for use in either Create
// or Update operations.
func makeJSONBody(recordPtr interface{}) (io.Reader, error) {
	f := getFields(recordPtr)
	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	jsonstr := fmt.Sprintf(`{"fields": %s}`, b)
	body := strings.NewReader(jsonstr)
	return body, nil
}

func getFields(ptr interface{}) interface{} {
	return reflect.ValueOf(ptr).Elem().FieldByName("Fields").Interface()
}

func getID(ptr interface{}) string {
	return reflect.ValueOf(ptr).Elem().FieldByName("ID").String()
}

func debugLog(s string, a ...interface{}) {
	fmt.Printf("DEBUG: "+s+"\n", a...)
}
