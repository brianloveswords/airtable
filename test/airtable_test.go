package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/brianloveswords/airtable"
)

func TestClientResource(t *testing.T) {
	client := airtable.Client{
		APIKey: os.Getenv("AIRTABLE_TEST_KEY"),
		BaseID: os.Getenv("AIRTABLE_TEST_BASE"),
	}
	main := client.NewResource("Main")

	// main.Int("Rating")
	// main.String("Name")
	// main.String("Notes")
	// main.StringArray("Cats")

	record, err := main.Get("recfUW0mFSobdU9PX", nil)

	fmt.Println("err:", err)
	fmt.Println("record:", record)

	t.Skipf("skipping")
}
