all:	chan-docs
	go install -ldflags="-X main.version=$$(git describe) -X main.commit=$$(git rev-parse HEAD) -X main.date=$$(date +%FT%H:%M:%S.%N)" ./...

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

# A demonstratio of using a Go plug-in to load a MySQL driver at
# runtime for use in a Plax test that uses a SQL channel to talk to
# MySQL.
#
# (This test will likely fail due to a timeout when trying to talk to
# MySQL.)
.PHONY: mysql
mysql:
	cd chans/sqlc/mysql && make
	cd cmd/plax && go install
	plax -test demos/mysql.yaml -log debug
