# The `plaxrun` Manual

## Table of Contents

- [The `plaxrun` Manual](#the-plaxrun-manual)
  - [Table of Contents](#table-of-contents)
  - [Using `plaxrun`](#using-plaxrun)
    - [Running](#running)
    - [Writing a Specification](#writing-a-specification)
      - [General properties](#general-properties)
      - [Tests Definition Section](#tests-definition-section)
      - [Test Groups Section](#test-groups-section)
        - [Test reference(s)](#test-references)
        - [Group reference(s)](#group-references)
        - [Test Group Parameters](#test-group-parameters)
        - [Iteration](#iteration)
        - [Guards](#guards)
      - [Parameters definition section](#parameters-definition-section)
      - [Reports definition section](#reports-definition-section)
    - [Running the example tests](#running-the-example-tests)
    - [Output](#output)
    - [Logging](#logging)
  - [References](#references)


## Using `plaxrun`

### Running

To get help, run:

`plaxrun -h`

```
Usage of plaxrun:
  -I value
    	YAML include directories
  -dir string
    	Directory containing test files (default ".")
  -g value
    	Groups to execute: Test Group Name
  -json
    	Emit JSON test output; instead of JUnit XML
  -labels string
    	Labels for tests to run
  -log string
    	Log level (info, debug, none) (default "info")
  -p value
    	Parameter Bindings: 
  -priority int
    	Test priority (default -1)
  -redact
    	enable redactions when -log debug (default true)
  -run string
    	Filename for test run specification (default "spec.yaml")
  -s string
    	Suite name to execute; -t options represent the tests in the suite to execute
  -t value
    	Tests to execute: Test Name
  -v	Verbosity (default true)
  -version
    	Print version and then exit
```

Use `-run` to specify the the path to the test run specification file

Use `-dir` to specify the path to the root of the test files directory

To run a single test group, use -g once:

`plaxrun -run cmd/plaxrun/demos/fullrun.yaml -dir demos -g basic`

To run multiple test groups, specify -g multiple times:

`plaxrun -run cmd/plaxrun/demos/fullrun.yaml -dir demos -g basic -g inclusion`

To run a single test or set of tests, specify -t:

`plaxrun -run cmd/plaxrun/demos/fullrun.yaml -dir demos -t basic`

To run a set of tests in a test suite, specify `-s` (plaxrun suite name references) and `-t` (plax test name references):

`plaxrun -run cmd/plaxrun/demos/fullrun.yaml -dir demos -s demos -t basic -t test-wait`

*Note:* A combination of `-g` an `-t` is allowed unless `-s` is used

Use `-json` to output a JSON representation of the test results instead of the Junit XML format.  This output includes `test.State` as the key `State` for each test case.

Use `-labels` [string] to set the labels filter for tests to run

Use `-priority` [int] to set the priority of tests to run

Use `-p 'PARAM=VALUE'` to pass bindings on the command line. You can specify `-b` multiple times:

`plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait-prompt -p '?WAIT=600' -p '?MARGIN=200'`

### Writing a Specification
A plaxrun specification is a `.yaml` file which contains the following major elements:

- `name` - The name of the test specification prefixed to all output test suite names
- `version` - The version of the test specification appended to the name
- `tests` - The set of defined tests referenced in test groups
- `groups` - The set of defined test groups referenced from other groups or the command line `-g` option(s)
- `params` - The set of parameters to be bound via shell command execution if values are not already bound via `-p` option(s) 

Here is an [example specification file](../cmd/plaxrun/demos/waitrun.yaml)

Let's start by breaking it down:

#### General properties
The general properties provided to uniquely identify test run executions

```yaml
name: waitrun
version: 0.0.1
```

- `name: waitrun` of the  specification which serves as the prefix of the test output suite name along with the `version`
- `version: 0.0.1` of the specification which serves as the the prefix of the test output suite name following the `name`

#### Tests Definition Section
The `tests:` section defines a set of tests, either test suites or test files, to be references by test groups.

```yaml
tests:
  wait:
    path: test-wait.yaml
    version: github.com/Comcast/plax
    params:
      - 'WAIT'
      - 'MARGIN'
```

- `wait:` is the test name used to reference the test from a test group
  - `path:` is the relative path to the test directory (Suite) or file (Test) based on `.` or the `-dir` option
  - `version: github.com/Comcast/plax` represents the name of the module that implements the plax plugin compatible with the plax execution engine test syntax.  This is optional if the default version `github.com/Comcast/plax` is being targeted
  - `params:` is the list of parameter name dependencies referencing the parameters defined in the `params` section.  All listed parameters will be evaluated for parameter binding values
    - `- 'WAIT'` is a parameter required by the `test-wait.yaml` test
    - `- 'MARGIN'` is a parameter required by the `test-wait.yaml` test

#### Test Groups Section
The `groups:` section defines a set of test groups which organize tests and nested test groups for execution.

##### Test reference(s)
Test groups can references defined tests as follows:
```yaml
groups:
  wait-prompt:
    tests:        
      - name: wait
```
 
- `wait-prompt:` is the group name. It is used to reference the test group from other test groups and the `-g` option; It shows how to define tests that likely require a prompt to enter tests parameter values not already bound
  - `params:` is the set of parameter key/value binding for the test group
    - `'WAIT': 600` is a parameter bound with the value `600`
    - `'MARGIN': 600` is a parameter bound with the value `200`
  - `tests` is the list of test references where the test `name` matches a test name defined in the `tests` section; each test is executed in sequence
    - `name: wait` is a test `name` reference to a test named `wait`
  
##### Group reference(s)
Test groups can reference nested test groups that have been defined as follows:
```yaml
 wait-group:
    groups:
      - name: wait-prompt
      - name: wait-no-prompt
      - name: wait-iterate
```
- `wait-group`: is the group name. it is used to reference the test group from other test groups and the `-g` option. It shows how to group test groups together inside a single test group
  - `groups:` is the list of group references where the group `name` matches a group name defined in the `groups` section; each group is executed in sequence
    - `name: wait-prompt` is a group `name` reference to a group named `wait-prompt`
    - `name: wait-no-prompt` is a group `name` reference to a group named `wait-no-prompt`
    - `name: wait-iterate` is a group `name` reference to a group named `wait-iterate`

##### Test Group Parameters
Test groups can have parameter bindings inherited by the referenced tests and groups as follows:
```yaml
  wait-no-prompt:
    params:
      'WAIT': 600
      'MARGIN': 200
    tests:        
      - name: wait
```
- `wait-no-prompt:` is the group name. It is used to reference the test group from other test groups and the `-g` option;  It shows how to define tests that do not required a prompt by binding all the necessary parameters with values
  - `params:` is the set of parameter key/value binding for the test group
    - `'WAIT': 600` is a parameter bound with the value `600`
    - `'MARGIN': 600` is a parameter bound with the value `200`
  - `tests:` is the list of test references where the test `name` matches a test name defined in the `tests` section; each test is executed in sequence
    - `name: wait` is a test `name` reference to a test named `wait`

##### Iteration
Test groups can iterate over a the referenced tests and groups as follows:
```yaml
  wait-iterate:
    iterate:
      params: |
        [
          {
            "WAIT": 300,
            "MARGIN": 100
          },
          {
            "WAIT": 600,
            "MARGIN": 200
          },
          {
            "WAIT": 900,
            "MARGIN": 300
          }
        ]
    tests:
      - name: wait
```

- `wait-iterate`: is the group name.  It is used to reference the test group from other test groups and the `-g` option.  It shows how to iterate through a set of tests or test groups
  - `iterate:` is the instruction to iterate over the test(s) or group(s) reference by this group
    - `params:` is the set of parameter object (key/value) bindings for each iteration of the test group
    - `'WAIT': 600` is a parameter bound with the value `600`
    - `'MARGIN': 600` is a parameter bound with the value `200`
  - `tests:` is the list of test references where the test `name` matches a test name defined in the  `tests` section; each test is executed in sequence
    - `name: wait` is a test `name` reference to a test named `wait`

##### Guards
Test groups can define a guard against test, group, or iteration execution.
```yaml
groups:
  wait-guard-test:
    iterate:
      dependsOn:
        - WAIT_LIST
        - ILLEGAL_WAIT
      params: '{WAIT_LIST}'
    tests:
      - name: wait
        guard:
          src: |
            return bs["WAIT"] != bs["ILLEGAL_WAIT"];

  wait-guard-group:
    groups:
      - name: wait-prompt
        guard:
          dependsOn:
            - DO_NOT_PROMPT
          libraries:
            - include/libs/boolean.js
          src: |
            return isFalse(bs["DO_NOT_PROMPT"]);
      - name: wait-no-prompt
      - name: wait-iterate

  wait-guard-iterate:
    iterate:
      dependsOn:
        - WAIT_LIST
        - ILLEGAL_WAIT
      params: '{WAIT_LIST}'
      guard:
          src: |
            return bs["WAIT"] != bs["ILLEGAL_WAIT"];
    tests:
      - name: wait
```
  - `guard:` is the instruction for a guard
    - `dependsOn:` evaluate the list of defined parameter references
    - `libraries:` import the listed Javascript libraries
    - `src:` execute the Javascript code to evaluate the guard; must return boolean [true|false]
#### Parameters definition section
The `params:` parameter definition section defines the parameter names to be bound to a value or set of values returned by a shell command.

Each parameter is composed of the following parts:

  - `envs:` is the section that defines the environment variables as mapped key/value pairs
  - `redact: [true|false]` is an optional flag to redact output of the parameter binding in the logs
  - `cmd:` is the command to execute.  `bash` makes for a great command execution script environment
  - `args:` are the arguments to pass to the command

An example set of parameters follows:

```yaml
params:
  'WAIT':
    include: include/commands/prompt.yaml
    envs:
      PROMPT: Enter wait
      DEFAULT: 300
  'MARGIN':
    include: include/commands/value.yaml
    envs:
      DEFAULT: 100
  'MARGIN':
    include: include/commands/value.yaml
    envs:
      DEFAULT: universe
    redact: true
```
- `'WAIT':` is the first parameter name
  - `include: include/commands/prompt.yaml` is the macro inclusion of the `prompt.yaml`) command meant for re-use by other parameter bindings
  - `envs:` is the section that defines the environment variables for the `prompt.yaml` command
      - `PROMPT:` is an optional environment variable for the prompt command; provides a default prompt
      - `DEFAULT:` is an optional environment variable for the prompt command; no value by default
- `'MARGIN':` is the second parameter name
  - `include: include/commands/value.yaml` is the macro inclusion of the `value.yaml` command meant for re-use by other parameter bindings
  - `envs:` is the section that defines the environment variables for the `value.yaml` command
    - `DEFAULT: 100` is a required environment variable for the `value.yaml` command if the environment variable is not already set
- `'WORLD':` is the third parameter name
  - `include: include/commands/value.yaml` is the macro inclusion of the `value.yaml` command meant for re-use by other parameter bindings
  - `envs:` is the section that defines the environment variables for the `value.yaml` command
    - `DEFAULT: 100` is a required environment variable for the `value.yaml` command if the environment variable is not already set
  - `redact: true` sets the flag to redact the output of the value of the parameter
  
The commands supported by plaxrun are greatly extensible.  Each command, a simple include file for reusability across parameters, is typically composed as follows:

```yaml
cmd: bash
args:
  - -c
  - |
    if [ -n "${!KEY}" ]; then
      echo $KEY=${!KEY}
    else
      : "${VALUE:?Variable is not set or empty}"

      echo $KEY=$VALUE
    fi
```

Within a command there is a preset environment variable call `KEY`.  `$KEY` is a reference to the name of the default parameter to which a value should be bound.  `${!KEY}` references the current value bound to `$KEY`.  The script should output to `stdout`, in this case via `echo` the key/value pair for the binding, e.g. `echo $KEY=$VALUE`.  The script can also echo out an other key/value pairs to bind more than one parameter value.  e.g. `echo MYPARAM="My Value"`

The current built in commands are:

- `command.yaml` returns the existing environment variable value or executes the given command to fetch the value
- `csv.yaml` returns an array of JSON objects with key and values mapped from the input CSV
- `prompt.yaml` returns the existing environment variable value or prompt the user for a command with an optional default value
- `value.yaml` returns the existing environment variable value or returns a default value

*Note:* Each command has a different set of required or optional environment variables (`envs`).  See each respective command `.yaml` file for additional information.

More commands can easily be added by plaxrun specification authors, e.g. fetch secure parameter values from Vault or invoke AWS CLI commands and bind the results to a parameter.

#### Reports definition section
The `reports:` definition section defines the report plugins to be executed to submit the result of the test execution.  Currently Plaxrun supports the
following report plugin types:

1. stdout - a report plugin that writes the test results to standard output in either XML or JSON format
   
Each report is composed of the following parts:

  - `config:` is the section that defines the report plugin specific configuration settings.  The config section supports parameter binding substitution.
  
An example report definition is as follows:
```yaml
params:
  STDOUT_REPORT_TYPE:
      include: include/commands/value.yaml
      envs:
        VALUE: JSON

reports:
  stdout:
    config:
      type: {STDOUT_REPORT_TYPE}
```

- `stdout:` is the default report plugin that reports to standard output
  - `config:` is the plugin specific configuration settings
    - `type: ` is the stdout report plugin output format.  `XML` is the default.  The example shows overriding to use `JSON` using the `{STDOUT_REPORT_TYPE}` parameter binding.

### Running the example tests
To run the test specifications described above:

- The following command runs just the `wait-no-prompt` test group
  ```
  plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait-no-prompt -json | jq .
  ```
- The following command runs just the `wait-prompt` test group
  ```
  plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait-prompt -json | jq .
  ```
- The following command runs both the `wait-no-prompt` and `wait-prompt` test group
  ```
  plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait-no-prompt -g wait-prompt -json | jq .
  ```
- The following command runs the `wait` test group which combines the `wait-no-prompt` and `wait-prompt` test groups
  ```
  plaxrun -run cmd/plaxrun/demos/waitrun.yaml -dir demos -g wait -json | jq .
  ```
### Output

After test execution, `plaxrun` will output results in JUnit XML:

```xml
<testsuite tests="1" failures="0" errors="0">
  <testcase name="tests/discovery-1.yaml" status="executed" time="11"></testcase>
</testsuite>
```

`plaxrun` will generate a suite name.

Use `-json` to output a JSON representation of test result objects.
This output includes the following for each test case:

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

### Logging

The `-log` command-line option accepts `none` (default), `info`, and
`debug`.  To provide some level of logging without printing some
possibly sensitive information, `info` does not log payloads or bindings
values.  In contrast, `-debug` will by default log binding values and
payloads.  However, with `-log debug`, the flag `-redact` enables some
log redactions:

1. Values with binding names that start with `X_` (ignoring
   non-alphabetic prefix characters like `?`) will be redacted from
   `debug` log output if `-redact=true` (default is true).
   If the redacted `X_` values are needed for debugging, use `-redact=false` 
   locally.

1. In a test, Javascript (usually executed via a `run` step) can add
   redactions using the functions `redactRegexp` and `redactString`.
   See documentation above for usage information.
   
See [`demos/redactions.yaml`](../demos/redactions.yaml) for an example
of both techniques.


## References

1. [The `plax` manual](manual.md)

