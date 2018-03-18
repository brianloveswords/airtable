clean:
	rm -f test/snapshots/*.snapshot

snapshot-test:
	go test -v test/airtable_snapshot_test.go

snapshot-test-update:
	go test -v test/airtable_snapshot_test.go -update
