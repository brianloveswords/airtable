package airtable

import "fmt"

// Rating type
type Rating int

// Text type
type Text string

// LongText type
type LongText string

// AttachmentThumbnail type
type AttachmentThumbnail struct {
	URL    string  `json:"url"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// AttachmentThumbnails type
type AttachmentThumbnails struct {
	Small AttachmentThumbnail `json:"small"`
	Large AttachmentThumbnail `json:"large"`
}

// Attachment type
type Attachment []struct {
	ID         string               `json:"id"`
	URL        string               `json:"url"`
	Filename   string               `json:"filename"`
	Size       float64              `json:"size"`
	Type       string               `json:"type"`
	Thumbnails AttachmentThumbnails `json:"thumbnails"`
}

// Checkbox type
type Checkbox bool

// MultipleSelect type
type MultipleSelect []string

// Date type
type Date string

// FormulaResult can be a string, number or error so leave it up to
// the user to parse
type FormulaResult struct {
	Number *float64
	String *string
	Error  *string
}

// UnmarshalJSON tries to figure out if this is an error, a string or
// a number.
func (f *FormulaResult) UnmarshalJSON(b []byte) error {
	fmt.Println("should unmarshal:", b)
	return nil
	// switch v := (*i).(type) {
	// case string:
	// 	f.String = &v
	// case float64:
	// 	f.Number = &v
	// case map[string]interface{}:
	// 	err, ok := v["error"].(string)
	// 	if !ok {
	// 		panic("parse error")
	// 	}
	// 	f.Error = &err
	// default:
	// 	panic("couldn't parse")
	// }
}

// Value returns the underlying value if the formula results is a
// string or a number, otherwise return nil pointer and false
func (f *FormulaResult) Value() (v interface{}, ok bool) {
	if f.Error != nil {
		return nil, false
	}
	if f.String != nil {
		return *f.String, true
	}
	return *f.Number, true
}

// RecordLink type
type RecordLink []string

// SingleSelect type
type SingleSelect string
