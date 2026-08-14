.PHONY: test integration

# Unit tests. No external dependency.
test:
	go test ./...

# Integration tests. Needs the staging database, which is provisioned by the
# pipeline and reached through the dcode-testdb helper. Nothing in this
# repository starts it.
integration:
	dcode-testdb up
	go test -tags integration ./integration/...
