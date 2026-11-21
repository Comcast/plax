# Recent changes

## cmd channel can read long lines

The cmd channel used `bufio.Scanner` for subprocess stdout and stderr,
which fails with `token too long` on lines larger than 64KiB. It now
uses `bufio.Reader.ReadLine` and waits to reap the process until those
readers finish, so a long line is not truncated when `Wait` would
otherwise close the pipes.

## `recv` topic actually considered

Due to a bug, a `recv` topic, if given, was not considered correctly.
In some cases, a match could result from a message with a topic not
equal to the given topic.  The current code fixes that bug.  If you
have a test that now fails and didn't previously, check if you are
specifying a topic for each `recv`.  In those cases, either adjust the
topic or remove the topic specification and check the message topic in
a guard (via `msg`, which has a value like
`{"topic":TOPIC,"payload":PAYLOAD}`).
