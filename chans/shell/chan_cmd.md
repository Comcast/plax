## `cmd`

This channel forwards messages to a shell's stdin, and messages
written to the shell's stdout and stderr are emitted.

### Options


1. `name` (string) is an opaque string used is reports about this
    Process.

1. `command` (string) is the name of the program.
    
    Subject to expansion.

1. `args` ([]string) is the list of command-line arguments for the program.
    
    Subject to expansion.

