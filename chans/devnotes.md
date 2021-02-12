## Registering


If you add a `_` import for your package in `chans/std/std.go`, then
the `plax` and `plaxrun` executables will get your channel type
registered automatically.


## Generating docs

To have a channel type Markdown documentation generated automatically,
define a test function starting with `TestDocs` in your `_test.go`,
and make that function call `DocSpec().Write()` on your channel type.

The build system will run these tests and then collect output with
filenames `chan_*.md`.  The method `DocSpec.Write()` writes a file
like that.  Example `TestDocs` function:

```Go
func TestDocs(t *testing.T) {
	(&SQS{}).DocSpec().Write("sqs")
}
```
