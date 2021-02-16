## `mother`

Mother ('mother') can make channels, and Mother is itself a
Channel.

### Input

Every MotherRequest will get exactly one MotherResponse.

1. `make` 

    1. `name` (string) is the requested name for the channel to be created.

    1. `type` is something like 'mqtt', 'httpclient', or 'sqs' (the
        types that are registered with a (or The) ChannelRegistry).

    1. `config` is the configuration (if any) for the requested channel.

### Output


1. `request` is the request the provoked this response.

    1. `make` 

        1. `name` (string) is the requested name for the channel to be created.

        1. `type` is something like 'mqtt', 'httpclient', or 'sqs' (the
            types that are registered with a (or The) ChannelRegistry).

        1. `config` is the configuration (if any) for the requested channel.

1. `success` (bool) reports whether the request succeeded.

1. `error` (string) not zero, is an error message for a failed
    request.

