package airtable

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type MainTestRecord struct {
	When        Date `from:"When?"`
	Rating      Rating
	Name        Text
	Notes       LongText
	Attachments Attachment
	Check       Checkbox
	Animals     MultipleSelect
	Cats        RecordLink
	Formula     FormulaResult
}

func TestClientResource(t *testing.T) {
	client := makeClient()

	id := "recfUW0mFSobdU9PX"

	var main MainTestRecord
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

	file, err := os.OpenFile(
		filepath.Join("testdata", "output.gob"),
		os.O_CREATE|os.O_RDWR, 0644)
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

	var expect MainTestRecord
	err = dec.Decode(&expect)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(main, expect) {
		t.Fatal("expected things to be equal")
	}
}
