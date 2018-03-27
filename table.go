package airtable

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"
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

func getFields(e *interface{}) (interface{}, error) {
	fields := reflect.ValueOf(*e).FieldByName("Fields")
	if !fields.IsValid() {
		return nil, errors.New("getFields: missing Fields")
	}
	if fields.Kind() != reflect.Struct {
		return nil, errors.New("getFields: Fields not a struct")
	}
	return fields.Interface(), nil
}

func getID(e *interface{}) (string, error) {
	id := reflect.ValueOf(*e).FieldByName("ID")
	if !id.IsValid() {
		return "", errors.New("getID: missing ID")
	}
	if id.Kind() != reflect.String {
		return "", errors.New("getID: ID not a string")
	}
	return id.String(), nil
}

const updateMethod = "PATCH"

// Update ...
func (t *Table) Update(record interface{}) error {
	f, err := getFields(&record)
	if err != nil {
		return err
	}
	id, err := getID(&record)
	if err != nil {
		return err
	}
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	body := strings.NewReader(fmt.Sprintf(`{"fields": %s}`, b))
	url := path.Join(t.name, id)
	_, err = t.client.RequestWithBody(updateMethod, url, Options{}, body)
	if err != nil {
		return err
	}
	return nil
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
