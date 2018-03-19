package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/brianloveswords/airtable"
)

type MainRecord struct {
	When        airtable.Date `from:"When?"`
	Rating      airtable.Rating
	Name        airtable.Text
	Notes       airtable.Text
	Attachments airtable.Attachment
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

	var main MainRecord
	mainReq := client.NewResource("Main", &main)

	mainReq.Get("recfUW0mFSobdU9PX", nil)

	fmt.Print(main)

	t.Skipf("skipping")
}
