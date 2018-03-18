package main

import (
	"os"
	"testing"

	"github.com/brianloveswords/airtable"
)

func TestClientResource(t *testing.T) {
	client := airtable.Client{
		APIKey: os.Getenv("AIRTABLE_TEST_KEY"),
		BaseID: os.Getenv("AIRTABLE_TEST_BASE"),
	}

	main := client.NewResource("Main", airtable.Record{
		"Rating":      airtable.Rating{},
		"Name":        airtable.Text{},
		"Notes":       airtable.Text{},
		"Attachments": airtable.Attachment{},
		"Check":       airtable.Checkbox{},
		"Animals":     airtable.MultipleSelect{},
		"When":        airtable.Date{},
		"Formula":     airtable.FormulaResult{},
		"Cats":        airtable.RecordLink{},
	})

	main.Get("recfUW0mFSobdU9PX", nil)

	// fmt.Println("err:", err)

	t.Skipf("skipping")
}
