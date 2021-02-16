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

