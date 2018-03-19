package airtable

// Rating ...
type Rating int

// Text ...
type Text string

// LongText ...
type LongText string

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
type Attachment []struct {
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

// FormulaResult can be a string, number or error so leave it up to
// the user to parse
type FormulaResult struct {
	Number *float64
	String *string
	Error  *string
}

// SelfParse ...
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

// Value ...
func (f *FormulaResult) Value() (interface{}, bool) {
	if f.Error != nil {
		return nil, false
	}
	if f.String != nil {
		return *f.String, true
	}
	return *f.Number, true
}

// RecordLink ...
type RecordLink []string

// SingleSelect ...
type SingleSelect string

// SelfParser ...
type SelfParser interface {
	SelfParse(v *interface{})
}
