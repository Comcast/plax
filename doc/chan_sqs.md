## `sqs`

In this implementation, message and subscription topics are
ignored.

### Options

For now, the target queue URL is provided when the channel is
created.  Eventually perhaps the queue URL could be the
message/subscription topic.

1. `Endpoint` (string) is optional AWS service endpoint, which can be
    provided to point to a non-standard endpoint (like a local
    implementation).

1. `QueueURL` (string) is the target SQS queue URL.

1. `DelaySeconds` (int64) is the publishing delay in seconds.
    
    Defaults to zero.

1. `VisibilityTimeout` (int64) is the default timeout for a message
    reappearing after a receive operation and before a delete
    operation.  Defaults to 10 seconds.

1. `MaxMessages` (int) is the maximum number of message to request.
    
    Defaults to 1.

1. `DoNotDelete` (bool) turns off automatic message deletion upon receipt.

1. `BufferSize` (int) is the size of the underlying channel buffer.
    Defaults to DefaultChanBufferSize.

1. `MsgDelaySeconds` (bool) enables extraction of property DelaySeconds
    from published message's payload, which should be a JSON of
    an map.
    
    This hack means that a test cannot specify DelaySeconds for
    a payload that is not a JSON representation of a map.
    ToDo: Reconsider.

1. `WaitTimeSeconds` (int64) is the SQS receive wait time.
    
    Defaults to one second.

