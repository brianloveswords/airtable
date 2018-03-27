test:
	go test -v ./...

clean:
	rm -f testdata/*

cover:
	go test -coverprofile=coverage.out

cover-html:
	go test -coverprofile=coverage.out &&\
	go tool cover -html=coverage.out
