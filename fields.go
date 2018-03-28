package airtable

import (
	"encoding/json"
	"log"
)

// Attachment type. When creating a new attachment, only URL and
// optionally Filename should be provided.
type Attachment []struct {
	ID         string               `json:"id"`
	URL        string               `json:"url"`
	Filename   string               `json:"filename"`
	Size       float64              `json:"size"`
	Type       string               `json:"type"`
	Thumbnails attachmentThumbnails `json:"thumbnails"`
}

type attachmentThumbnail struct {
	URL    string  `json:"url"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type attachmentThumbnails struct {
	Small attachmentThumbnail `json:"small"`
	Large attachmentThumbnail `json:"large"`
}

// TODO: make MultiSelect more useful. It's a natural fit for a Set
// type, but we don't have sets out of the box and it currently seems
// frivolous to pull in an external dependency or even make a naive set
// type just for this.

// MultiSelect type. Alias for string slice.
type MultiSelect []string

// TODO: make RecordLink more useful. For example, if we know what table
// the record links are supposed to come from, we could automatically
// hydrate those links instead of returning strings. We could also
// automatically create new records when necessary if the linked record
// object is novel in a Create operation.

// RecordLink type. Alias for string slice.
type RecordLink []string

// FormulaResult can be a string, number or error.
type FormulaResult struct {
	Number *float64
	String *string
	Error  *string
}

// UnmarshalJSON tries to figure out if this is an error, a string or
// a number.
func (f *FormulaResult) UnmarshalJSON(b []byte) error {
	i := new(interface{})
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}
	switch v := (*i).(type) {
	case string:
		f.String = &v
	case float64:
		f.Number = &v
	case map[string]interface{}:
		err, ok := v["error"].(string)
		if !ok {
			panic("parse error")
		}
		f.Error = &err
	default:
		log.Fatal("couldn't parse Formula type as number, string or error")
	}
	return nil
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
