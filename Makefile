clean:
	rm -f test/snapshots/*.snapshot

test:
	go test -v ./...

snapshot-test:
	go test -v test/airtable_snapshot_test.go

snapshot-test-update:
	go test -v test/airtable_snapshot_test.go -update
