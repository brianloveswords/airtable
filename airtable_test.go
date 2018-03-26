package airtable

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/brianloveswords/wiretap"
)

var (
	update = flag.Bool("update", false, "update the tests")
	check  = flag.Bool("check", false, "check the value")
)

type MainTestRecord struct {
	ID          Text
	CreatedDate Date
	Fields      struct {
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
}

type LongListRecord struct {
	ID          Text
	CreatedDate Date
	Fields      struct {
		Auto    Autonumber `json:"autonumber"`
		Created Date       `json:"created"`
	}
}

func TestClientTableLongList(t *testing.T) {
	// we can't use the wiretap because the offsets are always different
	// TODO: ignore certain params from wiretap?
	client := makeDefaultClient()
	table := client.Table("Long")
	list := []LongListRecord{}

	options := Options{
		Sort: Sort{
			{"Auto", SortDesc},
		},
		Fields: []string{"Auto"},
	}

	if err := table.List(&list, options); err != nil {
		t.Fatal("expected table.List(...) err to be nil", err)
	}

	if len(list) < 200 {
		t.Fatalf("should have gotten 200+ results, got %d", len(list))
	}

	entry := list[0]

	expect := len(list)
	result := entry.Fields.Auto
	if int(result) != expect {
		t.Fatalf("expected first result to be %d, got %d", expect, result)
	}

	if entry.Fields.Created != "" {
		t.Fatalf("should not have gotten created field")
	}
}

func TestClientTableList(t *testing.T) {
	client := makeClient()
	table := client.Table("Main")
	list := []MainTestRecord{}
	if err := table.List(&list, Options{}); err != nil {
		t.Fatalf("expected table.List(...) err to be nil %s", err)
	}

	if *check {
		fmt.Printf("%#v\n", list)
		t.Skip("skipping...")
	}

	if len(list) == 0 {
		t.Fatalf("should have gotten results")
	}

	entry := list[0]

	if entry.Fields.Name == "" {
		t.Fatal("should have gotten a name from list results")
	}

	if entry.ID == "" {
		t.Fatal("should have found an ID")
	}
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

	if main.Fields.Name == "" {
		t.Fatal("should have gotten a name")
	}
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

type credentials struct {
	APIKey string
	BaseID string
}

func mustOpen(p string) io.Reader {
	file, err := os.Open(p)
	if err != nil {
		log.Fatal("could not open file", err)
	}
	return file
}

func loadCredentials() credentials {
	file := mustOpen("secrets.env")
	dec := json.NewDecoder(file)
	creds := credentials{}
	if err := dec.Decode(&creds); err != nil {
		log.Fatal("could not decode secrets.env", err)
	}
	return creds
}

func makeClient() *Client {
	tap := makeWiretap()
	creds := loadCredentials()
	return &Client{
		APIKey:     creds.APIKey,
		BaseID:     creds.BaseID,
		HTTPClient: tap.Client,
	}
}
func makeDefaultClient() *Client {
	creds := loadCredentials()
	return &Client{
		APIKey: creds.APIKey,
		BaseID: creds.BaseID,
	}
}

func fixturePath(t *testing.T, fixture string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("problems recovering caller information")
	}
	return filepath.Join(filepath.Dir(filename), "testdata", fixture)
}

func writeFixture(t *testing.T, fixture string, content []byte) {
	err := ioutil.WriteFile(fixturePath(t, fixture), content, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func loadFixture(t *testing.T, fixture string) string {
	content, err := ioutil.ReadFile(fixturePath(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func makeWiretap() *wiretap.Tap {
	store := wiretap.FileStore("testdata")
	var tap wiretap.Tap
	if *update {
		tap = *wiretap.NewRecording(store)
	} else {
		tap = *wiretap.NewPlayback(store, wiretap.StrictPlayback)
	}
	return &tap
}
