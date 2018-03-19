package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/brianloveswords/airtable"
)

var (
	update = flag.Bool("update", false, "update the tests")
	check  = flag.Bool("check", false, "check the value")
)

type MainRecord struct {
	When        airtable.Date `from:"When?"`
	Rating      airtable.Rating
	Name        airtable.Text
	Notes       airtable.LongText
	Attachments airtable.Attachment
	Check       airtable.Checkbox
	Animals     airtable.MultipleSelect
	Cats        airtable.RecordLink
	Formula     airtable.FormulaResult
}

func TestClientResource(t *testing.T) {
	client := airtable.Client{
		APIKey: os.Getenv("AIRTABLE_TEST_KEY"),
		BaseID: os.Getenv("AIRTABLE_TEST_BASE"),
	}

	id := "recfUW0mFSobdU9PX"

	var main MainRecord
	mainReq := client.NewResource("Main", &main)
	mainReq.Get(id, nil)

	if *check {
		fmt.Println(main)

		if v, ok := main.Formula.Value(); ok {
			switch v.(type) {
			case string:
				fmt.Println("it's a string")
			case float64:
				fmt.Println("it's a float")
			}
		}

		t.Skip("skipping...")
	}

	file, err := os.OpenFile("output.gob", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}

	if *update {
		enc := gob.NewEncoder(file)
		enc.Encode(main)
		t.Skip("skipping...")
	}

	file.Seek(0, 0)
	dec := gob.NewDecoder(file)

	var expect MainRecord
	err = dec.Decode(&expect)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(main, expect) {
		t.Fatal("expected things to be equal")
	}
}
