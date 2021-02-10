all:
	go install ./...

install: all

.PHONY: unit-tests
unit-tests:
	go test -v ./...

test: unit-tests

.PHONY: demos
demos: plax-demos plaxrun-demos

plax-demos: all
	plax -dir demos -labels selftest

plaxrun-demos: all
	plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait-test-group

