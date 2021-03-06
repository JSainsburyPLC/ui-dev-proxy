build:
	go build -o ui-dev-proxy main.go

test:
	go vet ./...
	go test -race -short ./...
	bash -c 'diff -u <(echo -n) <(gofmt -s -d .)'

fmt:
	go fmt ./...

release-dry-run:
	goreleaser --snapshot --skip-publish --rm-dist

release:
	goreleaser --rm-dist
