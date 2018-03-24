package airtable

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/brianloveswords/wiretap"
)

var (
	update = flag.Bool("update", false, "update the tests")
	check  = flag.Bool("check", false, "check the value")
)

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
