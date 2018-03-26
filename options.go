package airtable

import (
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strings"
)

// SortType indicates which direction to sort the results in.
type SortType string

// List of sort types
const (
	SortDesc = "desc"
	SortAsc  = "asc"
)

// Options ...
type Options struct {
	Sort   Sort
	Offset string
	Fields []string
	typ    reflect.Type
}

// Sort ...
type Sort [][2]string

// Encode ...
func (o Options) Encode() string {
	q := []string{}

	if o.Offset != "" {
		q = append(q, "offset="+e(o.Offset))
	}

	// This creates encoded version of something like this:
	// "sort[0][field]=Name&sort[0][direction]=desc". It will look up
	// the JSON tag on the related field in the struct passed in to
	// hold the response. If there's no JSON tag, it uses the raw
	// field name.
	if len(o.Sort) != 0 {
		for i, sort := range o.Sort {
			field, direction := getFieldName(sort[0], o.typ), sort[1]
			sortstr := fmt.Sprintf("%s=%s&%s=%s",
				e(fmt.Sprintf("sort[%d][field]", i)),
				e(field),
				e(fmt.Sprintf("sort[%d][direction]", i)),
				e(direction),
			)
			q = append(q, sortstr)
		}
	}

	if len(o.Fields) != 0 {
		for i, name := range o.Fields {
			field := getFieldName(name, o.typ)
			fieldstr := fmt.Sprintf("%s=%s",
				e(fmt.Sprintf("fields[%d]", i)),
				e(field),
			)
			q = append(q, fieldstr)
		}
	}

	query := strings.Join(q, "&")
	return query
}

func getFieldName(n string, t reflect.Type) string {
	field := n

	fields, _ := t.FieldByName("Fields")
	f, ok := fields.Type.FieldByName(n)
	if !ok {
		log.Fatalf("could not sort by %s: no such field in %s", field, t.String())
	}
	if json, ok := f.Tag.Lookup("json"); ok {
		field = json
	}
	return field
}

func e(s string) string {
	return url.QueryEscape(s)
}
