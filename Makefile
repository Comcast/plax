all:
	cd cmd/plax && go install
	cd cmd/plaxrun && go install
	cd cmd/yamlincl && go install

.PHONY: unit-tests
units-tests:
	cd dsl && go test
	cd chans && go test
	cd invoke && go test

.PHONY: demo-tests
demo-tests: plax-demos plaxrun-demos

plax-demos: all
	plax -dir demos -labels selftest

plaxrun-demos: all
	plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait-test-group

test: unit-tests demo-tests
