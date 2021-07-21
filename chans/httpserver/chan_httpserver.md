## `httpserver`

An HTTPServer will emit the requests it receives from HTTP clients,
and the server should receive the responses to forward to those
clients.

To use this channel, you first 'recv' a client request, and then
you 'pub' the response, which the Chan will forward to the HTTP
client.

Note that you have to do 'pub' each specific response for each
client request.

### Options


1. `host` (string) 

1. `port` (int) 

1. `parsejson` (bool) 

### Input

1. `path` (string) 

1. `form` (url.Values) is the parsed form values.

1. `headers` (map[string][]string) is the map from header name to header values.

1. `method` (string) is the HTTP request method.

1. `body` (interface {}) is the request body (if any).
    
    This body is parsed as JSON if ParsedJSON is true.

1. `error` (string) is a generic error message (if any).

### Output

1. `headers` (map[string][]string) is the map from header name to header values.

1. `body` (interface {}) is the response body.

1. `statuscode` (int) 

1. `serialization` (*dsl.Serialization) is the serialization used to make a string
    representation of the body.

