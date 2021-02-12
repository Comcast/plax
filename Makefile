all:	chan-docs
	go install ./...

install: all

test: unit-tests

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

