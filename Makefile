clean:
	rm -f test/snapshots/*.snapshot

test:
	go test -v ./...

update-test:
	go test -v ./... -update

check-test:
	go test -v ./... -check
