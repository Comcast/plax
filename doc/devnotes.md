# Developer notes

## Releases

We're trying [`goreleaser`](https://goreleaser.com/).

To make and publish a new release:

1. Install [`goreleaser`](https://goreleaser.com/).

1. Figure out the tag for the release.  Consider using
   [`svu`](https://github.com/caarlos0/svu). Example `svu patch`.

1. Create and push the tag:

    ```Shell
	git tag -a v0.5.7 -m "Endgame: Fix all remaining bugs"
	git push origin v0.5.7 # ?
	```

1. Run `make release`.


## Implementing a new channel type

### Checklist

1. Create a new directory in [`chans`](../chans).
1. Implement `dsl.Chan`.
1. Write good Go doc comments that work well with the hacky Markdown
   documentation generation.
1. Provide a `TestDocs` Go test that calls `.DocSpec().Write()`.
1. Update [`chans/std/std.go`](../chans/std/std.go) to import your package.

### Discussion

Your implementation should implement `dsl.Chan`.  See "Generating
channel docs" below for some background on writing Go doc comments
that work well with generating Markdown docs.

You should include a Go test like

```Go
func TestDocs(t *testing.T) {
	(&MQTT{}).DocSpec().Write("mqtt")
}
```

The `Makefile` will run these tests and collect the `Markdown` that
`.DocSpec().Write` will hopefully generate.

Updating [`chans/std/std.go`](../chans/std/std.go) makes `plax` and
`plaxrun` get all of the standard packages registered.


## Generating channel docs

We're currently experimenting with a hack to generate Markdown channel
type docs (example: [`chan_mqtt.md`](chan_mqtt.md)) from Go source
comments and reflection.  See [`../dsl/docspec.go`](../dsl/docspec.go)
for the shameless tactics involved.

This code generation considers the Go doc comment for the channel
type.  The first sentence is dropped in the Markdown.  See
[`chan_mqtt.md`](chan_mqtt.md) and
[`../chans/mqtt/mqtt.go`](../chans/mqtt/mqtt.go) for an example.

## `Msg.Payload` type

In the beginning `Msg.Payload` was an `interface{}`.  For the most
part, that approach worked okay since almost all of our messages were
structured (using JSON for serialization).  As we expanded our
targets, non-structured payloads introduced ambiguity in how messages
were interpretered for pattern matching vs regular expression
matching.  We moved to message payload strings, and we introduced
explicit serializations (with JSON as the default), which eliminated
most of those ambiguities.

