package airtable

import (
	"encoding/json"
	"fmt"
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
func (c *Client) Table(name string, record interface{}) Table {
	// TODO: panic early if record is not a pointer
	return Table{
		client: c,
		name:   name,
		record: record,
	}
}

// Get returns information about a resource
func (r *Table) Get(id string, options QueryEncoder) (*GetResponse, error) {
	fullid := path.Join(r.name, id)
	bytes, err := r.client.RequestBytes("GET", fullid, options)
	if err != nil {
		return nil, err
	}

	var res GetResponse
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return nil, err
	}

	// record comes in as an `interface {}` so let's get a pointer for
	// it and unwrap until we can get a value for the underlying struct
	refPtrToStruct := reflect.ValueOf(&r.record).Elem()
	structAsInterface := refPtrToStruct.Interface()
	refStruct := reflect.ValueOf(structAsInterface).Elem()
	refStructType := refStruct.Type()

	for i := 0; i < refStruct.NumField(); i++ {
		f := refStruct.Field(i)
		fType := refStructType.Field(i)

		key := fType.Name
		if from, ok := fType.Tag.Lookup("json"); ok {
			key = from
		}

		if v := res.Fields[key]; v != nil {
			switch f.Kind() {
			case reflect.Struct:
				handleStruct(key, &f, &v)
			case reflect.Bool:
				handleBool(key, &f, &v)
			case reflect.Int:
				handleInt(key, &f, &v)
			case reflect.Float64:
				handleFloat(key, &f, &v)
			case reflect.String:
				handleString(key, &f, &v)
			case reflect.Slice:
				handleSlice(key, &f, &v)
			case reflect.Interface:
				handleInterface(key, &f, &v)
			default:
				panic(fmt.Sprintf("UNHANDLED CASE: %s of kind %s", key, f.Kind()))
			}
		}
	}
	return &res, nil
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
