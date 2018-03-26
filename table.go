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
func (r *Table) Get(id string, record interface{}) error {
	fullid := path.Join(r.name, id)
	bytes, err := r.client.RequestBytes("GET", fullid, nil)
	if err != nil {
		return err
	}
	recordType := reflect.TypeOf(record)
	responseType := reflect.StructOf([]reflect.StructField{
		{Name: "ID", Type: reflect.TypeOf("")},
		{Name: "Fields", Type: recordType},
		{Name: "CreatedTime", Type: reflect.TypeOf("")},
	})
	container := reflect.New(responseType)
	container.
		Elem().
		FieldByName("Fields").
		Set(reflect.ValueOf(record))

	return json.Unmarshal(bytes, container.Interface())
}

// List returns stuff
func (r *Table) List(listPtr interface{}, options QueryEncoder) error {
	bytes, err := r.client.RequestBytes("GET", r.name, options)
	if err != nil {
		return err
	}

	recordType := reflect.TypeOf(listPtr).Elem().Elem()
	entryType := reflect.StructOf([]reflect.StructField{
		{Name: "ID", Type: reflect.TypeOf("")},
		{Name: "Fields", Type: recordType},
		{Name: "CreatedTime", Type: reflect.TypeOf("")},
	})

	responseType := reflect.StructOf([]reflect.StructField{
		{Name: "Records", Type: reflect.SliceOf(entryType)},
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
		entry := recordList.Index(i).FieldByName("Fields")
		list = reflect.Append(list, entry)
	}
	reflect.ValueOf(listPtr).Elem().Set(list)
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
