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
	recordType := reflect.ValueOf(record).Type()
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

	err = json.Unmarshal(bytes, container.Interface())
	if err != nil {
		return err
	}
	return nil
}

// List returns stuff
func (r *Table) List(options QueryEncoder) (*ListResponse, error) {
	bytes, err := r.client.RequestBytes("GET", r.name, options)
	if err != nil {
		return nil, err
	}
	var res ListResponse
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return nil, err
	}

	return &ListResponse{}, nil
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
