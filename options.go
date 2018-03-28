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

// SortDesc and SortAsc are used in the Sort type to indicate the
// direction of the sort.
const (
	SortDesc = "desc"
	SortAsc  = "asc"
)

// Options is used in the Table.List method to adjust and control the response
type Options struct {
	// Sort the response. See the package example for usage usage
	Sort Sort

	// Which fields to include. Useful when you want to exclude certain
	// fields if you aren't using them to save on network cost.
	Fields []string

	// Maximum amount of record to return. If MaxRecords <= 100, it is
	// guaranteed the results will fit in one network request.
	MaxRecords uint

	// Formula used to filer the results. See the airtable formula
	// reference for more details on how to create a formula:
	// https://support.airtable.com/hc/en-us/articles/203255215-Formula-Field-Reference
	Filter string

	// Name of the view to use. If set, only the records in that view
	// will be returned. The records will be sorted and filtered
	// according to the order of the view.
	View string

	// Airtable API performs automatic data conversion from string
	// values if typecast parameter is passed in. Automatic conversion
	// is disabled by default to ensure data integrity, but it may be
	// helpful for integrating with 3rd party data sources.
	Typecast bool

	offset string
	typ    reflect.Type
}

// Sort represents a pair of strings: a field and a SortType
type Sort [][2]string

// Encode turns the Options object into a query string for use in URLs.
func (o Options) Encode() string {
	q := []string{}

	if o.offset != "" {
		q = append(q, "offset="+esc(o.offset))
	}

	if o.Typecast != false {
		q = append(q, "typecast=true")
	}

	if o.Filter != "" {
		q = append(q, "filterByFormula="+esc(o.Filter))
	}

	if o.View != "" {
		q = append(q, "view="+esc(o.View))
	}

	if o.MaxRecords != 0 {
		q = append(q, fmt.Sprintf("maxRecords=%d", o.MaxRecords))
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
				esc(fmt.Sprintf("sort[%d][field]", i)),
				esc(field),
				esc(fmt.Sprintf("sort[%d][direction]", i)),
				esc(direction),
			)
			q = append(q, sortstr)
		}
	}

	if len(o.Fields) != 0 {
		for i, name := range o.Fields {
			field := getFieldName(name, o.typ)
			fieldstr := fmt.Sprintf("%s=%s",
				esc(fmt.Sprintf("fields[%d]", i)),
				esc(field),
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

func esc(s string) string {
	return url.QueryEscape(s)
}
