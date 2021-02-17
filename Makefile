all:	chan-docs
	go install ./...

install: all

test: unit-tests chan-docs

.PHONY: unit-tests
unit-tests:
	go test -v ./...

.PHONY: demos
demos: plax-demos plaxrun-demos

.PHONY: chan-docs
chan-docs:
	go test -run=Doc ./...
	find . -path ./doc -prune -false -o -name 'chan_*.md' -exec mv \{\} doc \; -print

plax-demos: all
	plax -dir demos -labels selftest

plaxrun-demos: all
	plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait-test-group

.PHONY: clean
clean:
	find . -name '*~' -exec rm \{\} \;
	rm -rf dist

.PHONY: dist
dist: clean
	goreleaser release --skip-publish --rm-dist

.PHONY: release
release: clean
	goreleaser release --rm-dist
