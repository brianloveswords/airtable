package airtable

import (
	"encoding/json"
	"path"
	"reflect"
)

// Table ...
type Table struct {
	name   string
	client *Client
	record interface{}
}

// Table returns a new table
func (c *Client) Table(name string) Table {
	// TODO: panic early if record is not a pointer
	return Table{
		client: c,
		name:   name,
	}
}

// Get returns information about a resource
func (t *Table) Get(id string, record interface{}) error {
	fullid := path.Join(t.name, id)
	bytes, err := t.client.RequestBytes("GET", fullid, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, record)
}

// List returns stuff
func (t *Table) List(listPtr interface{}, options *Options) error {
	if options == nil {
		options = &Options{}
	}

	oneRecord := reflect.TypeOf(listPtr).Elem().Elem()
	options.typ = oneRecord

	bytes, err := t.client.RequestBytes("GET", t.name, options)
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

// ListResponse contains the response from listing records
type ListResponse struct {
	Records []GetResponse
	Offset  string
}

// GetResponse contains the response from requesting a resource
type GetResponse struct {
	ID          string
	Fields      map[string]interface{}
	CreatedTime string
}
