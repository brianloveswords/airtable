package main_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/brianloveswords/airtable"
)

var (
	update = flag.Bool("update", false, "update the tests")
)

func TestClientRequestBytes(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		resource string
		options  func() airtable.QueryEncoder
		snapshot string
		notlike  string
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
			options: func() airtable.QueryEncoder {
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
			options: func() airtable.QueryEncoder {
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
			options: func() airtable.QueryEncoder {
				q := make(url.Values)
				q.Add("fields", "[this will make it fail]")
				return q
			},
			testerr: func(err error) bool {
				_, ok := err.(airtable.ErrClientRequestError)
				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := airtable.Client{
				APIKey: os.Getenv("AIRTABLE_TEST_KEY"),
				BaseID: os.Getenv("AIRTABLE_TEST_BASE"),
			}

			var options airtable.QueryEncoder
			if tt.options != nil {
				options = tt.options()
			}

			output, err := client.RequestBytes(tt.resource, options)
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

func fixturePath(t *testing.T, fixture string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("problems recovering caller information")
	}
	return filepath.Join(filepath.Dir(filename), "snapshots", fixture)
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
