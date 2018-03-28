package airtable_test

import (
	"fmt"

	"github.com/brianloveswords/airtable"
)

func ExampleNewRecord() {
	type BookRecord struct {
		airtable.Record
		Fields struct {
			Title  string
			Author string
			Rating int
			Tags   airtable.MultiSelect
		}
	}

	binti := &BookRecord{}
	airtable.NewRecord(binti, airtable.Fields{
		"Title":  "Binti",
		"Author": "Nnedi Okorafor",
		"Rating": 4,
		"Tags":   airtable.MultiSelect{"sci-fi", "fantasy"},
	})

	fmt.Println(binti.Fields.Author)
	// Output: Nnedi Okorafor
}
func ExampleNewRecord_withoutNewRecord() {
	// You can avoid using NewRecord if you use a named struct instead
	// of an anonymous struct for the Fields field in record struct.

	type Book struct {
		Title  string
		Author string
		Rating int
		Tags   airtable.MultiSelect
	}

	type BookRecord struct {
		airtable.Record
		Fields Book
	}

	binti := &BookRecord{
		Fields: Book{
			Title:  "Binti",
			Author: "Nnedi Okorafor",
			Rating: 4,
			Tags:   airtable.MultiSelect{"sci-fi", "fantasy"},
		},
	}

	fmt.Println(binti.Fields.Author)
	// Output: Nnedi Okorafor
}
