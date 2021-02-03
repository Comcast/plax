# The Plax Manual

> To vouch this, is no proof, without more wider and more overt test.

--[Shakespeare, _Othello_](http://shakespeare.mit.edu/othello/othello.1.3.html)

## Table of Contents

- [The Plax Manual](#the-plax-manual)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
  - [Using Plax](#using-plax)
    - [Running](#running)
      - [Plax](#basic-use)
	  - [Plaxrun](#using-plaxrun)
    - [Writing Tests](#writing-tests)
      - [Channel types](#channel-types)
      - [Including YAML in other YAML](#including-yaml-in-other-yaml)
      - [Name](#name)
      - [Labels](#labels)
      - [Priority](#priority)
      - [Documentation strings](#documentation-strings)
      - [Negative](#negative)
      - [Retries](#retries)
      - [Bindings](#bindings)
      - [String commands](#string-commands)
      - [Channels](#channels)
      - [Javascript libraries](#javascript-libraries)
      - [Circuit breaker](#circuit-breaker)
      - [Pattern matching](#pattern-matching)
      - [Specifications](#specifications)
	- [Output](#output)
  - [References](#references)
  
## Installation

To build Plax, you need [Go installed](https://golang.org/doc/install).

Once you have a working `go` environment, you can install `plax` using
the following commands from the root of this repository.

```shell
go get github.com/Comcast/plax/...
```

Check that you can execute `plax`:

## Using Plax

### Basic use
```shell
plax -h
```

```
Usage of plax:
  -I value
    	YAML include directories
  -check-string-subst string
    	perform string-based substitution and exit
  -check-struct-subst string
    	perform structured substitution and exit
  -dir string
    	Directory containing test specs
  -error-exit-code
    	Return non-zero on any test failure
  -json
    	Emit docs suitable for indexing with plaxdb
  -labels string
    	Optional list of required test labels
  -list
    	Show report of known tests; don't run anything.  Assumes -dir.
  -log string
    	log level (info, debug, none) (default "info")
  -network string
    	net.Dial network to deal to force IPv4
  -p value
    	Parameter values: PARAM=VALUE
  -priority int
    	Optional lowest priority (where larger numbers mean lower priority!); negative means all (default -1)
  -retry string
    	Specify retries: number or {"N":N,"Delay":"1s","DelayFactor":1.5}
  -seed int
    	Seed for random number generator
  -test string
    	Filename for test specification (default "test.yaml")
  -test-suite string
    	Name for JUnit test suite (default "{TS}")
  -v	Verbosity (default true)
  -version
    	Print version and then exit
```


The `plax` command can run a single test or a set of test.

To run a single test, use `-test`:

```shell
plax -test demos/simple.yaml -log debug
```

To run tests in a directory, use `-dir DIRNAME`.  When using `-dir`,
you can also specify a comma-separated list of required labels with
`-labels`.  For example, to run tests that are labeled `foo` or `bar`:

```shell
plax -dir demos -labels selftest
```

You can also specific a minimum priority via `-priority`.  For
example:

```shell
plax -dir demos -priority 3 -labels selftest
```

will run all `selftest` tests in the `demos` directory that that a
priorty _less than or equal to_ 3.

You can pass bindings in the command line using `-p`.  You can specify
multiple `-p` values:

```shell
plax -test foo.yaml -p '?!WANT=tacos' -p '?!N=3'
```


### Using `plaxrun`

For more sophisticated Plax test execution, see the [`plaxrun`
manual](plaxrun.md), which documents using `plaxrun` to run lots of
Plax tests under various configurations.


### Writing Tests

You write a test specification in
[YAML](https://en.wikipedia.org/wiki/YAML).  This section describes
the structure of a test specification.  Also see [these
examples](../demos).  [`basic.yaml`](demos/basic.yaml) is a good,
small example of a test specification.

#### Channel types

A Plax test does I/O using "channels".  Currently Plax supports the
following channel types:

1. `mock`: A demo channel that's useful for stand-alone tests that do
   not do any actual I/O.  This channel just echos what is `pub`ed to
   it.
   
1. `cmd`: A subprocess that receives messages via its `stdin` and that
   emits messages via its `stdout`.

1. `mqtt`: An MQTT client.  Configuration:

	1. `BrokerURL` is the URL for the MQTT broker.  This required
       value has the form `PROTOCOL://HOST:PORT`.

    1. `CertFile` is the optional filename for the client's certificate.

	1. `CACertFile` is the optional filename for the certificate
        authority.

    1. `KeyFile` is the optional filename for the client's private key.

	1. `Insecure` will given the value for the `tls.Config
        InsecureSkipVerify`.  Controls whether a client verifies the
        server's certificate chain and host name. If
        InsecureSkipVerify is true, crypto/tls accepts any certificate
        presented by the server and any host name in that
        certificate. In this mode, TLS is susceptible to
        machine-in-the-middle attacks unless custom verification is
        used. This should be used only for testing.

    1. `ALPN` specifies the [Application-Layer
	   ProtocolNegotiation](https://en.wikipedia.org/wiki/Application-Layer_Protocol_Negotiation)
	   for the connection.  For example, see
	   https://docs.aws.amazon.com/iot/latest/developerguide/protocols.html.

	1. `AuthorizerName`, `Token`, `TokenSig`, and `TokenHeader` are
	   optional values that are useful for authentication using an
	   [the AWS IoT Core customer
	   authorizer](https://docs.aws.amazon.com/iot/latest/developerguide/custom-authorizer.html).
	   When Token is not empty, then you should probably also provide
	   `AuthorizerName` and `TokenSig`.

	    `TokenHeader` is the name of the header which will have value
        given by `Token`, and `TokenSig` is the signature for the
        `Token` value.

	    `AuthorizerName` is the value for the header
        `x-amz-customauthorizer-name`, which is used when making a AWS
        IoT Core WebSocket connection.

	1. `BufferSize` specifies the capacity of the internal Go channel.
   		The default is [../chans/mqtt.go](`DefaultMQTTBufferSize`).

	1. `PubTimeout` is the timeout in milliseconds for MQTT PUBACK.

	1. `SubTimeout` is the timeout in milliseconds for MQTT SUBACK.

	1. `ClientID` is the MQTT client id.

	1. `Username` is the optional MQTT client username.

	1. `Username` is the optional MQTT client password.

	1. `CleanSession`, when true, will not resume a previous MQTT
		session for this client id.

	1. `WillEnabled`, if true, will establish an MQTT Last Will and
       Testament.  See `WillTopic`, `WillPayload`, `WillQoS`, and
       `WillRetained`.

	1. `WillTopic` gives the MQTT LW&T topic.  See `WillEnabled`.

	1. `WillPayload` gives the MQTT LW&T payload.  See `WillEnabled`.
	
	1. `WillQoS` specifies the MQTT LW&T QoS.  See `WillEnabled`.
	
	1. `WillRetained` specifies the MQTT LW&T "retained" flag.  See `WillEnabled`.

	1. `KeepAlive` is the duration in seconds that the MQTT client
		should wait before sending a PING request to the broker.

	1. `PingTimeout` is the duration in seconds that the client will
	    wait after sending a PING request to the broker before
	    deciding that the connection has been lost.  The default is 10
	    seconds.

	1. `ConnectTimeout` is the duration in seconds that the MQTT
     	client will wait after attempting to open a connection to the
     	broker.  A duration of 0 never times out.  The default 30
     	seconds.  This property does not apply to WebSocket
     	connections.

	1. `MaxReconnectInterval` specifies maximum duration in seconds
     	between reconnection attempts.

	1. `AutoReconnect` turns on automation reconnection attempts when
       a connection is lost.

	1. `WriteTimeout` is the duration to wait for a PUBACK.

	1. `ResumeSubs` enables resuming of stored (un)subscribe messages
       when connecting but not reconnecting if `CleanSession` is
       false.

1. `kds`: A primitive KDS channel.  Currently this channel only
   supports consuming from a Kinesis stream.
   
   The implementation is based on
   [`github.com/harlow/kinesis-consumer`](https://github.com/harlow/kinesis-consumer).
   
   The only option is `StreamName`, which should be the name of
   Kinesis stream to consume.  This channel will consume all shards
   starting with the most recent records.
   
1. `sqs`: A basic SQS consumer and publisher.  Options:

	1. `QueueURL`
	1. `DelaySeconds` is the publishing delay in seconds.
	1. `VisibilityTimeout` sets the receiving "visibility timeout" before a non-deleted message reappears in the queue.
	1. `MaxMessages` specifies the maximum number of messages to request at each poll.
	1. `DoNotDelete` prevents deletion of the message after receipt.
	1. `MsgDelaySeconds` enables extraction of DelaySeconds from published message's payload.
	1. `WaitTimeSeconds` is the SQS receive wait time (in seconds).  Defaults to one second.


1. `httpclient`: An HTTP client.  This channel type doesn't have any
   options.  To use a this channel, you `pub` a request, and then you
   `recv` the response.  See [this
   demo](../demos/http-client.yaml) for an example.
   
   You can either specify form values (via
   `form`) or the request body explicity (via `body`).  If the given
   `body` is structured (and not a string), that structure is
   JSON-serialized for you.
   
    The published payload should be a JSON object with the following
    properties:
	
	1. `method`: The HTTP request method. Default is `GET`.
	
	1. `url`: The requested URL (required).
	
	1. `headers`: An optional map of header names to _arrays_ of
       values.  Example:
	   
        ```{JSON}
		{"headers":{"someHeader":["someValue"]}}
		```
		
	1. `body`: Optional body for the request.
	
	1. `requestbodyserialization`: Either `JSON` or `string`, which
       specifies how to serialize the request body.  Defaults to
       "JSON".
	   
	1. `form`: Optional map of names to _arrays_ of values.  If you
       specify this property, then `body` becomes this URL-encoded
       value.
	   
	1. `responsebodyserialization`: Either `JSON` or `string`, which
       specifies how to deserialize the response body.  Defaults to
       "JSON".
	   
	1. `pollinterval`: An optional duration (in [Go
       syntax](https://golang.org/pkg/time/#ParseDuration)) to have
       this request repeat on that interval.  You can also specify an
       `Id`, which you can then use to `terminate` the polling.  If
       you don't specify an `id` along with a `pollinterval`, you can
       stil do `terminate: last`, which will always work to terminate
       the most recent polling HTTP request.  See [this
       demo](../demos/webdriver.yaml).

1. `httpserver`: An HTTP server channel type.  Configuration:

    1. `host`: name of the network interface to use
	1. `port`: number of port on which to listen
	1. `parsejson`: Boolean that indicates whether to parse received
       message bodies as JSON.  Defaults to `true`.
	   
    To use an `httpserver` channel, you `recv` each request that the
    HTTP listener receives from a client, and you `pub` the response
    to forward to the client.  The request your `recv` has a payload
    with `path`, `headers`, `method`, `body`, and `error` properties.
    The response can have `headers`, `body`, `statuscode`, and
    `serialization`. The `serialization` property, which can have a
    `JSON` (default) or `string` value, specifies how the given `body`
    is serialized before forwarding as the HTTP response body to the
    HTTP client.
	
	You can of course run both an `httpclient` and `httpserver` in a
    Plax test, and you can make those channels talk to each other via
    real HTTP.  (You can also `curl` your `httpserver` if you want,
    but remember you have to `recv` and `pub` to that channel for each
    actual HTTP request.)  See [`demos/http.yaml`](../demos/http.yaml)
    for an example.

As the needs arise, we can add channel types like:

1. KDS publisher
1. Kafka consumer and publisher

and so on.


#### Including YAML in other YAML

Plax supports including some YAML in other YAML.

<a name="includes"> Plax looks for:
  - `#include<FILENAME>`
  - `$include<FILENAME>`
  - `include: FILENAME`
  - `includes: [FILENAME-1, FILENAME-N]`
</a>

in your YAML.  When Plax
encounters one of these directives, Plax attempts to read `FILENAME`
from directories specified on the command line via `-I DIR`.  You can
specify multiple `-I DIR` arguments.  The test spec's current
directory is added to the end of the list of directories to search.
Then YAML that's read is substituted for the include directive.

For `include: FILENAME` or `includes: FILENAME` should be a YAML
representation of a map.  That map is added to the map that contained
the `include: FILENAME` property.

`$include<FILENAME>`, which must be a value in an array, results in a
splice into that array by the _array_ represented by the YAML in
`FILENAME`.  Unlike `cpp`, Plax looks for `FILENAME` relative to the
test's directory.

`#include<FILENAME>` is replaced by that value with the thing
represented by `FILENAME` in YAML.  Unlike `cpp`, Plax looks for
`FILENAME` relative to the test's directory.

The utility command `yamlincl` performs just this processing.  Example:


```Shell
cat demos/include.yaml | yamlincl -I demos
```


#### Name

The optional `name` field is used for giving a concise identifier for
a test.  The value isn't actually used for anything at the moment.

```yaml
name: discovery-1
```

#### Labels

The optional `label` field is used to list general attributes of or
tags for the test. For example, what components are tested or what
type of test it is.  A label can be any string, but we shouldn't go
crazy here.  The `plax` tool's `-label` option can run only tests that
that all of the tiven labels (separated by commas).  For example `plax
-dir tests -labels integration,happy-path` would run all the tests in
the directory `tests` that have labels `integration` and `happy-path`.

Example test labels:

```yaml
labels:
  - happy-path
  - integration
  - authentication
```

#### Priority

The optional `priority` field assigns a priority to the test. Priority
is used to select which tests to run.  Priority can be passed into
`plax` using the `-priority` option.  For example, when a priority of
`2` is passed into `plax`, it will run all level `1` and level `2`
priority tests, but not level `3`.

```yaml
priority: 1
```

#### Documentation strings

The optional `doc` attribute can provide documentation as a string.
Note that YAML supports [multi-line
strings](https://yaml-multiline.info/), and your `doc` value should
probably be one of those.

We might add a `links` attribute that could specify a list of URLs of
interest.


#### Negative

The optional `negative` field means a failure is interpreted as a
success, and a failure is interpreted as a success.  Errors (as
opposed to failures) are not affected.

Example:

```yaml
negative: true
```

#### Retries

The optional `retries` field specifies a retry policy:

1. `n`: The maximum number of retries
1. `delay`: The initial delay (in [Go
   syntax](https://golang.org/pkg/time/#ParseDuration))
1. `delayfactor`: A multiplier to applied to a delay to give the next
   delay.
   
The default behavior is no retry.

Example:

```yaml
retries:
  n: 3
  delay: 1s
  delayfactor: 2
```


#### Bindings 

Bindings allow a test to have values that change at runtime.  For
example, you could have a binding for a certifcate filename that would
allow you to run the same test with different filenames.

In an expression, bindings subsitution takes two forms: structured and
textual.

When processing a _string_, each occurance of `{B}`, where `B` is a
bound variable, is literally replaced by that variable's binding.  For
example, the string `"I like {?x}."` with with bindings
`{"?x":"queso"}` results in `"I like queso."`

When processing _structured data_ (which can be obtained implicitly
from a string that's legal JSON), bindings subsitution is itself
structured.  Only bindings starting with `?` are considered, and only
exact bindings are replaced.  For example, the object `{"need":"?x"}`
with bindings `{"?x":"chips"}` becomes `{"need":"chips"}`.

Note the difference between string-based bindings substitution and
structured bindings substitution.  The former results in a string
value while the latter results in a value with the type of whatever
the bindings value has.  Plax will warn if you do bindings
substitution in a string context where the binding value isn't itself
a string.

All of this substitution is called recursively until a fixed point is
reached, so you can go crazy with self-referencing substitutions.  See
[`demos/recursive-subst.yaml`](../demos/recursive-subst.yaml) for a
mild example.

If you've got a binding for a variable that you want to remove for
subsequent steps, you can use a `run` step to remove the binding
manually (say with `delete(test.Bindings["?x"])`).  See
[this](../demos/example-bindings-conflict.yaml),
[this](../demos/example-bindings-clear.yaml), and
[that](../demos/example-bindings-deconflict.yaml) example regarding
unintentional bindings substitutions.  In a `recv` step (see below),
you can also specify `clearbindings: true` to ignore any existing
bindings that do not start with `?!`.

See the end of the next section regarding the order of operations.

To provide a binding at runtime, use the `-p` flag:

```Shell
plax -test tests/this.yaml -p '?!CERT=that.pem' -p '?!KEY=key.pem'
```

These two bindings, for `?!CERT` and `?!KEY`, start with `?!` to
ensure that those bindings are not cleared when `clearbindings` is
specified in a `recv` step.  This `recv` behavior is described below.

When using `-p` to specify a binding, if the given value parses as
JSON, then that parsed value is used as the binding value.  This
behavior is convenient when doing structured binding substitution.


#### String commands

Several string values have special powers.

<a name="at-at-filename"></a>If one of these strings looks like
`@@FILENAME`, then Plax attempts to substitute the contents of the
file with name `FILENAME` for that string.  The file is read relative
to the directory that contained the test specification.

<a name="bang-bang-javascript"></a>If one of these string starts with
`!!`, then remainder of the string is executed as Javascript.
Bindings substitution applies.  The value returned by this Javascript
is substituted for string.

These string commands are processed in the order above: first `@@` and
then `!!`.  (So a file's contents could start with `!!`, which would
trigger Javascript execution.)  Bindings are substituted _after_
string processing.  All of this substitution is called recursively
until a fixed point is reached, so you can drive yourself crazy with
self-referencing substitutions.

The documentation below mentions when a string has these special
powers ("string commands").  Most strings have these powers.


#### Channels

A Plax test can work with multiple "channels" simulatenously.  A
channel is something can that do I/O, and an MQTT client is the
classic example.  We can also have channels for a KDS consumer, a KDS
publisher, an HTTP client, an SQS consumer, and so on.

See [Channel types](#channel-types) for a summary of available types.

There is one primoridal channel named `mother`.  You can ask `mother`
to make other channels for you by publishing (`pub`) a message to
`mother`, who will always reply.  Example of making a request and
receiving the reply:

```YAML
- pub:
    doc: Please make a mock channel.
    chan: mother
    payload:
      make:
        name: mock
        type: mock
- recv:
    doc: Check that our request succeeded.
    chan: mother
    pattern:
      succeed: true
```

The payload of the request should specify the `name` for the channel
to be created, the `type` of the channel (e.g., `mock`, `mqtt`, `cmd`,
etc), and an optional `config` for any channel options.

Note that a test might want to verify that a request to `mother`
failed.  For example, a request to `mother` to create an MQTT client
with invalid credentials _should_ fail.  Authentication tests often
have this form.


#### Javascript libraries

A test can specify `libraries`, which should be a list of filenames.
Each file should contain Javascript.  All of those files are loaded
for _each_ Javascript execution.  Each file is read from the directory
that contains the test spec.

Example:

```YAML
libraries:
  - library.js
  - foo.js
```

That declaration will result in `library.js` and `foo.js` loaded
before each `run` or `guard`.

#### Circuit breaker

A test specification can specify `maxsteps`, which defaults to 100.
The test will fail if it takes more than this number of steps in
total.  This property is useful as a circuit breaker for an potential
loop caused by a `branch` step.

#### Pattern matching

In a receive (`recv`) step (describe below), the given `pattern` is
matched against incoming messages.  This matching is [Sheens message
pattern matching](https://github.com/Comcast/sheens#pattern-matching).
Here are some
[examples](https://github.com/Comcast/sheens/blob/master/match/match.md).
You can maybe use `go get github.com/Comcast/sheens/cmd/patmatch` to
experiment:

```Shell
patmatch -p '{"want":"?x"}' -m '{"want":"queso","when":"now"}'
```

#### Specifications

The `spec` field is where most of the action will take place.  Each
phase in the `phases` consists of one or more _steps_.  A step is a
single operation.  Currently the following steps are supported:

1. `sub`: Subscribe to a topic (filter).

    1. `chan`: The name for the channel for this step.
	
	1. `pattern`: The topic (or topic filter) for the subscription.
      	If the value is a JSON string, the string is first parsed as
      	JSON.  Parameters and bindings
      	[substitution](#substitutions) applies.

1. `recv`: Look for certain messages that have arrived. <a name="recv">

    1. `chan`: The name for the channel for this step.
	
    1. `topic`: Optional: The expected message should arrive on this
       topic.  Parameters and bindings
       [substitution](#substitutions) applies.
	   
	1. `schema`: An option URI for a JSON schema, which is then used
       to validate the in-coming message before any other processing.
		
	1. `serialization`: How to deserialize in-coming payloads. Either
       `string` or `JSON`, and `JSON` is the default.

    1. `pattern`: A _pattern_ that the message must match.  Parameters
       and bindings [substitution](#substitutions)
       applies.  [String commands](#string-commands) are also available
	
		The pattern has [this
        structure](https://github.com/Comcast/sheens#pattern-matching).
		
		All bindings for variables that start with `?*` are removed
        before this pattern substitution.
		
		Alternately, give a `regexp` instead of a `pattern`.
		
	1. `regexp`: A [regular expression](https://github.com/google/re2)
       (instead of a `pattern`).  A regular expression will probably
       be more convenient for receiving non-JSON input.
	   
	   A named group like `(?P<foo>.*)` match results in a new binding
       for a `?*foo`.  If the name starts with an uppercase rune as in
       `Foo`, then the variable will be `?Foo`.
	   
	   See [`demos/regexp.yaml`](../demos/regexp.yaml) for an example.
	
	1. `clearbindings`: If true, delete all `test.Bindings` for
       variables that do not start with `?!`.
	   
	1. `timeout`: Optional timeout in [Go
       syntax](https://golang.org/pkg/time/#ParseDuration).
	
	1. `target`: Target is an optional switch to specify what part of
       	the incoming message is considered for matching.
		
		By default, only the payload is matched.  If `target` is
       	"message", then matching is performed against
       	`{"Topic":TOPIC,"Payload":PAYLOAD}` which allows matching
       	based on the topic of in-bound messages.
		
	1. `guard`: <a href="https://en.wikipedia.org/wiki/Guard_(computer_science)">Guard</a>
	    is optional Javascript that should return a boolean to
	    indicate whether this `recv` has been satisfied.
		
	    Parameter and bindings
      	[substitution](#substitutions) applies, and
      	[string commands](#string-commands) are also available
	
       	<a name="javascript-failure"></a>The code is executed in a
	    function body, and the code should 'return' a boolean or an
	    expression of the form `Failure(STRING)`.  A boolean indicates
	    whether the `recv` will succeed.  A `Failure` will terminate
	    the test immediately as failed.
		
		The following variables are bound in the
	    global environment:
		
		1. `bindingss`: the set (array) of bindings returned by
            `match()`.
			
		1. `bindings` (also `bs`): The first set of bindings returned
           by `match()`.  Probably the only ones you care about.

        1. `elapsed`: the elapsed time in milliseconds since the
	        last step.

        1. `msg`: the receved message
            (`{"topic":TOPIC,"payload":PAYLOAD}`).

        1. `test`: The whole test object.
		
		    In particular, your Javascript code can use `test.State`,
			which is a map from strings to anythings.  You can use
			`test.State` to store data accessible by Javascript to be
			executed later.  Also `test.T`, which is the time the
			previous step executed, is also available.
			
			`test.Bindings` is a map from pattern variables (e.g.,
            `?foo`) to values.  The map is set after each successful
            `recv` pattern match to be the (first) set of bindings
            from that match.  This map is used to replace any pattern
            variables in the `payload` of the next `pub`.
			
			With great power comes great responsibility.
			
	    1. `print`: a function that prints its arguments to log
           output.
		
		1. `match`: [Sheen](https://github.com/Comcast/sheens)'s
            [pattern
            matching](https://github.com/Comcast/sheens#pattern-matching)
            function.
		   
		    ```Javascript
			BINDINGSS = match(PATTERN,MSG,BINDINGS);
			```
			
			1. `PATTERN` is a Javascript thing.
			1. `MSG` is a Javascript thing.
			1. `BINDINGS` is an Javascript object representing input
               bindings (often just `{}`).
			1. `BINDINGSS` is the _set_ of set of bindings returned by
               `match`.  (So that second `S` isn't really a typo.)
			   
			If an error occurs, it's thrown.
			
			See [`demos/match.yaml`](../demos/match.yaml) for an
            example.

	1. `run`: Executed Javascript just like `guard` except that the
       return value is ignored.  Parameters and bindings
       [substitution](#substitutions) applies.
       [String commands](#string-commands) are also available
	
1. `pub`: Publish a message.

    1. `chan`: The name for the channel for this step.
	
    1. `topic`: Optional: The expected message should arrive on this
       topic.  Parameters and bindings
       [substitution](#-substitutions) applies.
	   
	1. `serialization`: How to serialize the payload. Either `string`
       or `JSON`, and `JSON` is the default.

	1. `payload`: A _pattern_ that the message must match.  If the
      	value is a JSON string, the string is first parsed as JSON.
      	Parameters and bindings
      	[substitution](#substitutions) applies.
      	[String commands](#string-commands) are also available.
		
	1. `schema`: An option URI for a JSON schema, which is then used
       to validate the out-going message.
		
	1. `run`: Execute Javascript just like a `recv`'s `guard` except
       that the return value is ignored.  Parameters and bindings
       [substitution](#substitutions) applies.
       [String commands](#string-commands) are also available.

1. `wait`: Wait for the given number of milliseconds.

1. `kill`: Kill the step's channel ungracefully.

    1. `chan`: The name for the channel for this step.
	
1. `exec`: Run a command.  Structure is the same as for an `initially`
    command.  See [`exec.yaml`](../demos/exec.yaml) for a simple
    example.  Parameters and bindings
    [substitution](#substitutions) applies.
   
1. `reconnect`: Attempt to reconnect the channel (even if still connected).

    1. `chan`: The name for the channel for this step.

1. `run`: Execute Javascript as in a `recv`'s guard except that the
   return value is ignored. Parameters and bindings
   [substitution](#substitutions) applies.
   
1. `branch`: A fancy mechanism for (conditional) branching to another
   phase.  Parameters and bindings
   [substitution](#substitutions) applies.
   
    The value of a `branch` is Javascript code that should return the
    (name) of the next phase or the empty string (to continue with the
    current phase).
	
	Example:
	
	```YAML
	branch: |
	  return 0 < test.State["need"] ? "here" : "there";
	```
	
1. `goto`: Go to another phase.

1. `doc`: A documentation string for a step that's just that
   documentation string.  Doesn't actually do anything.

Most steps have an optional `chan` field, which should name the
channel for the step.  A spec can declare a `defaultchan` that will be
used for all steps.  If your test has only one channel, then that
channel is the default.

```yaml
spec:
  defaultchan: cpe
```

<a name="fails"></a>
You can also specify that a step is required to _fail_:

```yaml
spec:
  phases:
    one:
      steps:
      - pub:
          topic: want
          payload: '"queso"'
        fails: true
```

Note that `fail` is specified at the same level as the type of step
(`pub`, `recv`, etc.).


<a name="skip"></a> You can also specify that a step should be skipped by
specifying `skip: true` in the step.

```yaml
spec:
  phases:
    one:
      steps:
      - pub:
          topic: want
          payload: '"queso"'
        skip: true
```

Note that `skip` is specified at the same level as the type of step
(`pub`, `recv`, etc.).


How you organize phases and steps is up to you.

You can specify your first phase using `initialphase`, which defaults
to `phase1`:

```yaml
spec:
  ...
  initialphase: boot
```

You can specify one or more "final" phases that are executed after
the main test execution (starting with the initial phase) terminates
regardless of any error encountered.

```yaml
spec:
  ...
  finalphases:
    - cleanup1
	- cleanup1
```

These phases are executed in the given order regardless of any errors
each might enocounter.  Note that a "final" phase can `goto` another
phase.  In that case, that target phase should (probably) not be
included in the `finalphases` list.

See [`finally.yaml`](../demos/finally.yaml) for a short example.


### Output

After test execution, `plax` (or [`plaxrun`](plaxrun.md)) will by
default output results in JUnit XML:

```xml
<testsuite tests="1" failures="0" errors="0">
  <testcase name="tests/discovery-1.yaml" status="executed" time="11"></testcase>
</testsuite>
```

For `plax`, use `-test-suite NAME` to specify the suite's `name`.  For
`plaxrun` a suite name will be generated.

For `plax` and `plaxrun` use `-json` to output a JSON respresentation
of test result objects.  This output includes the following for each
test case:

```json
[
  {
    "Type": "suite",
    "Time": "2020-12-02T21:33:09.0728586Z",
    "Tests": 1,
    "Passed": 1,
    "Failed": 0,
    "Errors": 0
  },
  {
    "Name": "/.../plax/demos/test-wait.yaml",
    "Status": "executed",
    "Skipped": null,
    "Error": null,
    "Failure": null,
    "Timestamp": "2020-12-02T21:33:09.077102Z",
    "Suite": "waitrun-0.0.1:wait-no-prompt:wait",
    "N": 0,
    "Type": "case",
    "State": {
      "then": "2020-12-02T21:33:09.0781375Z"
    }
  }
]
```

## References

1. [The `plaxrun` manual](plaxrun.md)

1. [Sheens message pattern
   matching](https://github.com/Comcast/sheens#pattern-matching), and
   some
   [examples](https://github.com/Comcast/sheens/blob/master/match/match.md)
   
1. [Sheens](https://github.com/Comcast/sheens) could be
   [used](https://github.com/Comcast/sheens/tree/master/sio/siomq) to
   implement more complex tests and simulations
   
1. [TCL "expect"](https://en.wikipedia.org/wiki/Expect)

1. [YAML Wikipedia page](https://en.wikipedia.org/wiki/YAML) and YAML
   [multi-line strings](https://yaml-multiline.info/) in particular

