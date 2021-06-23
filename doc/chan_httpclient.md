## `httpclient`

This channel type implements HTTP requests.  A test publishes a
request that includes a URL.  This channel performs the HTTP
request and then forwards the response for the test to receive.

### Options

Currently this channel doesn't have any configuration.

### Input


1. `method` (string) is the usual HTTP request method (e.g., GET, POST).

1. `url` (string) is the target for the request.

1. `headers` (map[string][]string) is map of HTTP header names to values.

1. `body` (interface {}) is the request body.

1. `requestBodySerialization` (dsl.Serialization) specifies what serialization
    (if any) to perform on the request's body.
    
    Possible values are 'string' and 'json' (default).

1. `responseBodyDeserialization` (dsl.Serialization) specifies what deserialization
    (if any) to perform on the response's body.
    
    Possible values are 'string' and 'json' (default).

1. `form` (url.Values) can contain form values, and you can specify these
    values instead of providing an explicit Body.

1. `ctl` (chans.HTTPRequestCtl) is optional data for managing polling
    requests.

    
    1. `id` (string) is used to refer to this request when it has a polling
        interval.

    1. `pollInterval` (string) not zero, will cause this channel to
        repeated the HTTP request at this interval.
        
        The timer starts after the last request has completed.
        (Previously the timer fired requests at this interval
        regardless of the latency of the previous HTTP request(s).)
        
        Value should be a string that time.ParseDuration can parse.

    1. `terminate` (string) not zero, should be the Id of a previous polling
        request, and that polling request will be terminated.
        
        No other properties in this struct should be provided.

1. `insecure` (bool) if true will skip server credentials verification.

### Output


1. `statuscode` (int) is the HTTP status code returned by the HTTP server.

1. `body` (interface {}) is the either the raw body or parsed body returned by
    the HTTP server.
    
    The requests's ResponseBodyDeserialization determines if
    and how deserialization occurs.

1. `error` (string) describes a channel processing error (if any) that
    occured during the request or response.

1. `headers` (map[string][]string) contains the response headers from the HTTP server.

