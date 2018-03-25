package airtable

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"
)

type MainTestRecord struct {
	When        Date `json:"When?"`
	Rating      Rating
	Name        Text
	Notes       LongText
	Attachments Attachment
	Check       Checkbox
	Animals     MultipleSelect
	Cats        RecordLink
	Formula     FormulaResult
}

func TestClientTableList(t *testing.T) {
	client := makeClient()
	table := client.Table("Main")
	res, err := table.List(nil)
	if err != nil {
		t.Fatalf("expected table.List(...) err to be nil %s", err)
	}

	fmt.Println(res)
}

func TestClientTableGet(t *testing.T) {
	client := makeClient()

	id := "recfUW0mFSobdU9PX"

	var main MainTestRecord
	table := client.Table("Main")
	if err := table.Get(id, &main); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if *check {
		fmt.Printf("%#v\n", main)
		t.Skip("skipping...")
	}

	fmt.Printf("%#v\n", main)
}

func TestClientRequestBytes(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		resource string
		snapshot string
		notlike  string
		queryFn  func() QueryEncoder
		testerr  func(error) bool
	}{
		{
			name:     "no options",
			method:   "GET",
			resource: "Main",
			snapshot: "no-options.snapshot",
		},
		{
			name:     "field filter: only name",
			method:   "GET",
			resource: "Main",
			queryFn: func() QueryEncoder {
				q := make(url.Values)
				q.Add("fields[]", "Name")
				return q
			},
			snapshot: "fields-name.snapshot",
			notlike:  "no-options.snapshot",
		},
		{
			name:     "field filter: name and notes",
			method:   "GET",
			resource: "Main",
			queryFn: func() QueryEncoder {
				q := make(url.Values)
				q.Add("fields[]", "Name")
				q.Add("fields[]", "Notes")
				return q
			},
			snapshot: "fields-name_notes.snapshot",
			notlike:  "fields-name.snapshot",
		},
		{
			name:     "request error",
			method:   "GET",
			resource: "Main",
			queryFn: func() QueryEncoder {
				q := make(url.Values)
				q.Add("fields", "[this will make it fail]")
				return q
			},
			testerr: func(err error) bool {
				_, ok := err.(ErrClientRequestError)
				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := makeClient()

			var options QueryEncoder
			if tt.queryFn != nil {
				options = tt.queryFn()
			}

			output, err := client.RequestBytes(tt.method, tt.resource, options)
			if err != nil {
				if tt.testerr == nil {
					t.Fatal(err)
				}

				if !tt.testerr(err) {
					t.Fatal("error mismatch: did not expect", err)
				}
			}

			if tt.snapshot == "" {
				return
			}

			if *update {
				fmt.Println("<<updating snapshots>>")
				writeFixture(t, tt.snapshot, output)
			}

			actual := string(output)
			expected := loadFixture(t, tt.snapshot)
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("actual = %s, expected = %s", actual, expected)
			}

			if tt.notlike != "" {
				expected := loadFixture(t, tt.notlike)
				if reflect.DeepEqual(actual, expected) {
					t.Fatalf("%s and %s should not match", tt.snapshot, tt.notlike)
				}
			}
		})
	}
}
