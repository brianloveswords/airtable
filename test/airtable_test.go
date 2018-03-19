package main

import (
	"encoding/gob"
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/brianloveswords/airtable"
)

var (
	update = flag.Bool("update", false, "update the tests")
)

type MainRecord struct {
	When        airtable.Date `from:"When?"`
	Rating      airtable.Rating
	Name        airtable.Text
	Notes       airtable.Text
	Attachments []airtable.Attachment
	Check       airtable.Checkbox
	Animals     airtable.MultipleSelect
	Formula     airtable.FormulaResult
	Cats        airtable.RecordLink
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

	file, err := os.OpenFile("output.gob", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}

	if *update {
		enc := gob.NewEncoder(file)
		enc.Encode(main)
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
