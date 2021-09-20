# Plax: an engine for testing messaging systems

[![Go Reference](https://pkg.go.dev/badge/github.com/Comcast/plax.svg)](https://pkg.go.dev/github.com/Comcast/plax)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v1.4%20adopted-ff69b4.svg)](CODE_OF_CONDUCT.md)


> And with reference to the narrative of events, far from permitting
> myself to derive it from the first source that came to hand, I did
> not even trust my own impressions, but it rests partly on what I saw
> myself, partly on what others saw for me, **the accuracy of the
> report being always tried by the most severe and detailed tests
> possible**.

-- [Thucydides, _The History of the Peloponnesian
War_](http://classics.mit.edu/Thucydides/pelopwar.1.first.html)


## Summary

Plax is a test automation engine for messaging systems.  This engine
is [designed](chans) to perform integrated testing of MQTT messaging,
Kinesis streams, SNS traffic, SQS [trafic](demos/sqs.yaml), Kafka I/O,
HTTP APIs, [subprocesses](demos/shell.yaml), mobile
[apps](demos/webdriver.yaml) (via
[WebDriver](https://www.w3.org/TR/webdriver/)), and more.

An author of a test specifies a sequence of input and expected outputs
over a set of channels that are connected to external services.
Execution of the test verifies that the expected output occurred.

## Automated Build Status
[![Test](https://github.com/Comcast/plax/actions/workflows/test.yml/badge.svg)](https://github.com/Comcast/plax/actions/workflows/test.yml)
[![Tag](https://github.com/Comcast/plax/actions/workflows/tag.yml/badge.svg)](https://github.com/Comcast/plax/actions/workflows/tag.yml)
[![Release](https://github.com/Comcast/plax/actions/workflows/release.yml/badge.svg)](https://github.com/Comcast/plax/actions/workflows/release.yml)

## Command-line tools in this repo

1. [`plax`](cmd/plax): The test engine (and probably the reason you
   are here).  Documentation is [here](doc/manual.md).
1. [`plaxrun`](cmd/plaxrun): Tool to run lots of Plax tests with
   various configurations.  Documentation is [here](doc/plaxrun.md).
1. [`plaxsubst`](cmd/plaxsubst): Utility to test/use parameter
   substitution independently from other Plax functionality.  Related
   documenation is [here](subst/README.md).
1. [`yamlincl`](cmd/yamlincl): YAML include processor utility.
   Documenation is [here](doc/manual.md#including-yaml-in-other-yaml).


## Usage

Clone this repo and [install Go](https://golang.org/doc/install).
Then:

```Shell
(cd cmd/plax && go install)
# Run one simple test.
plax -test demos/mock.yaml -log debug
# Run several tests.
plax -dir demos -labels selftest
```

That last command runs all the [example test specs](demos) that are
labeled as `selftest`. [`basic.yaml`](demos/basic.yaml) is a good,
small example of a test specification.

## Plugins
1. [Octane plugin](doc/octane_plugin.md)

## The language

See the [main documentation](doc/manual.md) and the [examples](demos).


## References

1. [Plax manual](doc/manual.md) and the [`plaxrun`
   manual](doc/plaxrun.md)
1. [Sheens pattern
   matching](https://github.com/Comcast/sheens#pattern-matching) and
   some
   [examples](https://github.com/Comcast/sheens/blob/master/match/match.md)
1. [Sheens](https://github.com/Comcast/sheens), which could be
   [used](https://github.com/Comcast/sheens/tree/master/sio/siomq) to
   implement more complex tests and simulations
1. [TCL "expect"](https://en.wikipedia.org/wiki/Expect)

