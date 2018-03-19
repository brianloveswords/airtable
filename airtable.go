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
	"reflect"
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

/* Field Types */

// Rating ...
type Rating int

// Text ...
type Text string

// AttachmentThumbnail ...
type AttachmentThumbnail struct {
	// WARNING: if you add any new types, make sure to update
	// `handleAttachment` or look forward to panics!
	URL    string  `from:"url"`
	Width  float64 `from:"width"`
	Height float64 `from:"height"`
}

// AttachmentThumbnails ...
type AttachmentThumbnails struct {
	Small AttachmentThumbnail `from:"small"`
	Large AttachmentThumbnail `from:"large"`
}

// Attachment ...
type Attachment struct {
	// WARNING: if you add any new types, make sure to update
	// `handleAttachment` or look forward to panics!
	ID         string               `from:"id"`
	URL        string               `from:"url"`
	Filename   string               `from:"filename"`
	Size       float64              `from:"size"`
	Type       string               `from:"type"`
	Thumbnails AttachmentThumbnails `from:"thumbnails"`
}

// Checkbox ...
type Checkbox bool

// MultipleSelect ...
type MultipleSelect []string

// Date ...
type Date string

// FormulaResult ...
type FormulaResult struct {
	Number float64
	String string
	Error  string
}

// RecordLink ...
type RecordLink []string

// SingleSelect ...
type SingleSelect string

// GetResponse contains the response from requesting a resource
type GetResponse struct {
	ID          string                 `json:"id"`
	Fields      map[string]interface{} `json:"fields"`
	CreatedTime string                 `json:"createdTime"`
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
	m, ok := (*v).(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as struct", key))
	}
	sType := s.Type()
	for i := 0; i < sType.NumField(); i++ {
		f := s.Field(i)
		fType := sType.Field(i)
		key := fType.Name
		if from, ok := fType.Tag.Lookup("from"); ok {
			key = from
		}

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

func handleAttachment(key string, f *reflect.Value, v *interface{}) {
	err := fmt.Sprintf("PARSE ERROR: could not parse column '%s' as map(attachment)", key)

	// ugh ok, so we need to do this
	// interface{} ->
	//   []interface{} ->
	//     []map[string]interface{} ->
	//       []Attachment
	as, ok := (*v).([]interface{})
	if !ok {
		panic(err)
	}

	dst := make([]Attachment, len(as))
	for i, a := range as {
		m, ok := a.(map[string]interface{})

		if !ok {
			panic(err)
		}

		// construct the attachment by looping through the fields of
		// the struct, finding in the attachment map by the lookup key
		// from the tag and passing to the appropriate handler
		var attachment Attachment
		structType := reflect.TypeOf(attachment)
		structValue := reflect.ValueOf(&attachment).Elem()
		for j := 0; j < structType.NumField(); j++ {
			f := structValue.Field(j)
			fType := structType.Field(j)
			key := fType.Tag.Get("from")

			mapval := m[key]

			switch f.Interface().(type) {
			case string:
				handleString(key, &f, &mapval)
			case float64:
				handleFloat(key, &f, &mapval)
			case AttachmentThumbnails:
				handleStruct(key, &f, &mapval)

			// this should never happen. if it does there's a type
			// that was added to Attachment that didn't get reflected
			// in this switch statement.
			default:
				panic("UNHANDLED CASE: handleAttachment. contact package maintainer")
			}
		}

		dst[i] = attachment
	}
	f.Set(reflect.ValueOf(dst))
}
func handleBool(key string, f *reflect.Value, v *interface{}) {
	b, ok := (*v).(bool)
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as bool", key))
	}
	f.SetBool(b)
}
func handleStringSlice(key string, f *reflect.Value, v *interface{}) {
	s, ok := (*v).([]interface{})
	if !ok {
		panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as slice", key))
	}

	// TODO: is there a better way to do this?
	dst := make([]string, len(s))
	for i, v := range s {
		// shadows main slice
		str, ok := v.(string)
		if !ok {
			panic(fmt.Sprintf("PARSE ERROR: could not parse column '%s' as slice of strings", key))
		}
		dst[i] = str
	}
	f.Set(reflect.ValueOf(dst))
}

func handleFormulaResult(key string, f *reflect.Value, v *interface{}) {
	switch decoded := (*v).(type) {
	case float64:
		f.Set(reflect.ValueOf(FormulaResult{Number: decoded}))
	case string:
		f.Set(reflect.ValueOf(FormulaResult{String: decoded}))
	case map[string]interface{}:
		e, ok := decoded["error"].(string)
		if !ok {
			panic(fmt.Sprintf("PARSE ERROR: forumla error for '%s' is not string", key))
		}
		f.Set(reflect.ValueOf(FormulaResult{Error: e}))
	default:
		panic(fmt.Sprintf("PARSE ERROR: formula '%s' is not string, int, or error", key))
	}
}

// Get returns information about a resource
func (r *Resource) Get(id string, options QueryEncoder) (*GetResponse, error) {
	fullid := r.name + "/" + id
	bytes, err := r.client.RequestBytes(fullid, options)
	if err != nil {
		return nil, err
	}

	var resp GetResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	// record comes in as an `interface {}` so let's get a pointer for
	// it and unwrap until we can get a value for the underlying struct
	refPtrToStruct := reflect.ValueOf(&r.record).Elem()
	structAsInterface := refPtrToStruct.Interface()
	refStruct := reflect.ValueOf(structAsInterface).Elem()
	refStructType := refStruct.Type()

	for i := 0; i < refStruct.NumField(); i++ {
		f := refStruct.Field(i)
		fType := refStructType.Field(i)

		key := fType.Name
		if from, ok := fType.Tag.Lookup("from"); ok {
			key = from
		}

		if value := resp.Fields[key]; value != nil {
			switch f.Interface().(type) {
			case Text:
				handleString(key, &f, &value)
			case SingleSelect:
				handleString(key, &f, &value)
			case Date:
				handleString(key, &f, &value)
			case Rating:
				handleInt(key, &f, &value)
			case []Attachment:
				handleSlice(key, &f, &value)
				//handleAttachment(key, &f, &value)
			case Checkbox:
				handleBool(key, &f, &value)
			case RecordLink:
				handleStringSlice(key, &f, &value)
			case MultipleSelect:
				handleStringSlice(key, &f, &value)
			case FormulaResult:
				handleFormulaResult(key, &f, &value)
			default:
				panic(fmt.Sprintf("UNHANDLED CASE: %v, contact package maintainer", fType.Type))
			}
		}
	}
	return &resp, nil
}

// Resource ...
type Resource struct {
	name   string
	client *Client
	record interface{}
}

// NewResource returns a new resource manipulator
func (c *Client) NewResource(name string, record interface{}) Resource {
	// TODO: panic early if record is not a pointer
	return Resource{name, c, record}
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
