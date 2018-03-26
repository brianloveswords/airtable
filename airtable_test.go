package airtable

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/brianloveswords/wiretap"
)

var (
	record = flag.Bool("record", false, "wiretap new outgoing requests")
	check  = flag.Bool("check", false, "check the value")
)

type MainTestRecord struct {
	ID          string
	CreatedTime Date
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
	ID          string
	CreatedTime Date
	Fields      struct {
		Auto    Autonumber `json:"autonumber"`
		Created Date       `json:"created"`
	}
}

func TestOptions(t *testing.T) {
	client := makeClient()
	table := client.Table("Long")
	list := []LongListRecord{}

	options := Options{
		MaxRecords: 3,
		Sort:       Sort{{"Auto", SortAsc}},
		Fields:     []string{"Auto"},
		Filter:     `{autonumber} > 2`,
		View:       "odds",
	}

	if err := table.List(&list, &options); err != nil {
		t.Fatal("expected table.List(...) err to be nil", err)
	}

	// odds, maxrecords 3, autonumber >2: 3 5 7
	entry := list[len(list)-1]
	expect := 7
	result := entry.Fields.Auto

	if entry.Fields.Created != "" {
		t.Fatalf("should not have gotten created field")
	}

	if int(result) != expect {
		t.Fatalf("expected result to be %d, got %d", expect, result)
	}
}

func TestClientTableLongList(t *testing.T) {
	// we can't use the wiretap because the offsets are always different
	// TODO: ignore certain params from wiretap?
	client := makeDefaultClient()
	table := client.Table("Long")
	list := []LongListRecord{}
	options := Options{
		Sort: Sort{{"Auto", SortAsc}},
	}

	if err := table.List(&list, &options); err != nil {
		t.Fatal("expected table.List(...) err to be nil", err)
	}

	if len(list) < 200 {
		t.Fatalf("should have gotten 200+ results, got %d", len(list))
	}

	entry := list[0]
	expect := 1
	result := entry.Fields.Auto
	if int(result) != expect {
		t.Fatalf("expected first result to be %d, got %d", expect, result)
	}
}

func TestClientTableList(t *testing.T) {
	client := makeClient()
	table := client.Table("Main")
	list := []MainTestRecord{}
	if err := table.List(&list, nil); err != nil {
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

func makeWiretap() *wiretap.Tap {
	store := wiretap.FileStore("testdata")
	var tap wiretap.Tap
	if *record {
		tap = *wiretap.NewRecording(store)
	} else {
		tap = *wiretap.NewPlayback(store, wiretap.StrictPlayback)
	}
	return &tap
}
