# RFC: Fancier substitutions

## Background

Today in Plax you can substitute parameters in expressions.  We mostly
deal in JSON-serializable payloads here, but we also need to support
plain text.

Today you can also use YAML "includes" to embed some YAML into a large
YAML structure.  This YAML inclusion supports splicing into arrays and
maps.

This RFC generalizes both types of substitution.

## Fancier substitutions

This [RFC](https://en.wikipedia.org/wiki/Request_for_Comments) offers
fancier substitutions that allow for more control of how substitutions
behave.  In addition, this RFC attempts to demonstrate that YAML
inclusions are really just a form of parameter substitution.

The goal is to remain backwards-compatible.

## String substitutions

All subtitutions are based on single string that contains zero or more
substitution specifications.  In its most general form, a substitution
specification looks like this:

> `{`_VAR_`|`_PROC_`|`_SERIALIZATION_`}`

1. _VAR_ is the name of parameter.
1. _PROC_ is an optional processor that consumes the parameter value
   given by _VAR_ and emits an object.
1. _SERIALIZATION_ specifies how to render the object.

The `|` _PROC_ is optional, and `|` _SERIALIZATION_ essentially
defaults to `| json`.

If the specification (the stuff in the `{}` and including those
braces) is surrounded by double-quotes, then those quotes are ignored.
This behavior allows text that includes substitutions to still be
valid JSON.

## Variables

A variable can be a normal parameter name, which should exist as key
in the given parameters.

In addition, if a "variable" starts with `@`, the rest of the variable
should be a filename that can be found in an include path.  The
contents of this file are read, and the result is deserialized based
on the filename extension (e.g., `.yaml`, `.json`, `.txt`).  The
resulting object is then processed in the same manner as an object
found in parameters.

## Processors

Processors might turn out to be unhelpful, but the idea was too
tempting to ignore.

The specification of a processor looks like

> *PROCESSOR_TYPE* *SRC* ...

In this RFC, there are two processor types processors: Javascript and
[jq](https://github.com/itchyny/gojq).  The _SRC_ is either Javascript
or an `jq` expression according to *PROCESSOR_TYPE*.  When using the
Javascript processor, `$` is bound to the (structured) value given by
_VAR_.

## Serializations

The _SERIALIZATION_ specifies how to render the result:

1. `text`: Assuming the object is string, just use that string as is
   (no delimiting quotes).
1. `text$`: Assuming the object is an array of strings, join that
   array with a comma and then use that result literially (without any
   delimiting quotes). 
1. `trim`: Same as `text` but all leading and trailing whitespace is trimmed.
1. `json`: Serialize as JSON.
1. `json$`: Serialize an _array_ as JSON and splice in those
   elements without the delimiting `[` and `]`.
1. `json@`: Serialize an _object_ as JSON and splice in those
   key/value pairs without the delimiting `{` and `}`.
1. `yaml`: Serialize as YAML.
1. `yaml$`: Serialize an _array_ as YAML and splice in those
   elements without array-delimiting syntax.
1. `yaml@`: Serialize an _object_ as YAML and splice in those
   key/value pairs without map-delimiting syntax.

The `$` indicates array splicing, and the `@` indicates map splicing.

## Examples

See [`demo.sh`](demo.sh):

```
echo '{"deliver":"{?want}"}' | plaxsubst -p '?want="tacos"'
{"deliver":"tacos"}

echo 'I like {?want|text}.' | plaxsubst -p '?want="tacos"'
I like tacos.

echo '{"deliver":"{?want}"}' | plaxsubst -p '?want=["tacos","chips"]'
{"deliver":["tacos","chips"]}

echo '{"deliver":["beer","{?want|json$}"]}' | plaxsubst -p '?want=["tacos","chips"]'
{"deliver":["beer","tacos","chips"]}

echo '{"deliver":"{?want}","n":{?want | js $.length | json}}' | plaxsubst -p '?want=["tacos","chips"]'
{"deliver":["tacos","chips"],"n":2}

echo '{"deliver":"{?want | jq .[0] | json}"}' | plaxsubst -p '?want=["tacos","chips"]'
{"deliver":"tacos"}

echo 'The order: {?want|text$}.' | plaxsubst -p '?want=["tacos","chips"]'
The order: tacos,chips.

echo 'The first item: {?want|jq .[0]|text}.' | plaxsubst -p '?want=["tacos","chips"]'
The first item: tacos.

echo '{"deliver":{"chips":2,"":"{?want|json@}"}}' |
    plaxsubst -p '?want={"tacos":2,"salsa":1}' -check-json-in -check-json-out
{"deliver":{"chips":2,"salsa":1,"tacos":2}}

echo 'I want <?want|text>.' | plaxsubst -d "<>" -p '?want="tacos"'
I want tacos.

echo '{"deliver":"?want"}' | plaxsubst -bind -p '?want={"tacos":3}'
{"deliver":{"tacos":3}}

echo '{"deliver":"?want | jq .[0]"}' | plaxsubst -bind -p '?want=[{"tacos":3},{"queso":1}]'
{"deliver":{"tacos":3}}
```

## Structured substitutions

In an "object", a string of the form `?VAR` will be replaced by a
binding for `?VAR`.  (Note the lack of braces as delimiters.)  The
value for this binding will in general be an object, so no
serialization is required.  As a generalization, a string of form
`?VAR | jq ...` will be replaced by the result of evaluating the `jq`
expression with `?VAR`'s binding as input.

