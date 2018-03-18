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

func TestRawRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		resource string
		options  func() airtable.QueryEncoder
		snapshot string
		notlike  string
	}{
		{
			name:     "Dishes, no options",
			method:   "GET",
			resource: "Dishes",
			snapshot: "dishes.snapshot",
		},
		{
			name:     "Dishes, name",
			method:   "GET",
			resource: "Dishes",
			options: func() airtable.QueryEncoder {
				q := make(url.Values)
				q.Add("fields[]", "Name")
				return q
			},
			snapshot: "dishes_fields-name.snapshot",
			notlike:  "dishes.snapshot",
		},
		{
			name:     "Dishes, name and notes",
			method:   "GET",
			resource: "Dishes",
			options: func() airtable.QueryEncoder {
				q := make(url.Values)
				q.Add("fields[]", "Name")
				q.Add("fields[]", "Notes")
				return q
			},
			snapshot: "dishes_fields-name-notes.snapshot",
			notlike:  "dishes_fields-name.snapshot",
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

			output, err := client.Request(tt.resource, options)
			if err != nil {
				t.Fatal(err)
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
