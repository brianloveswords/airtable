package airtable

// Rating type
type Rating int

// Text type
type Text string

// LongText type
type LongText string

// AttachmentThumbnail type
type AttachmentThumbnail struct {
	URL    string  `from:"url"`
	Width  float64 `from:"width"`
	Height float64 `from:"height"`
}

// AttachmentThumbnails type
type AttachmentThumbnails struct {
	Small AttachmentThumbnail `from:"small"`
	Large AttachmentThumbnail `from:"large"`
}

// Attachment type
type Attachment []struct {
	ID         string               `from:"id"`
	URL        string               `from:"url"`
	Filename   string               `from:"filename"`
	Size       float64              `from:"size"`
	Type       string               `from:"type"`
	Thumbnails AttachmentThumbnails `from:"thumbnails"`
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

// SelfParse on FormulaResult figures out whether the result is a
// number, string, or error object.
func (f *FormulaResult) SelfParse(i *interface{}) {
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
		panic("couldn't parse")
	}
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
