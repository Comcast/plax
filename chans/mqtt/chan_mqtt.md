## `mqtt`

This channel impl talks MQTT to a broker.  The configuration
provides the data required to set up the connections.  Messages
published to this channel are forwarded to the MQTT broker, and
messages received from the MQTT broker are available for a test to
receive.

The topic of a message published to the channel becomes the MQTT
topic for the message.  Similarly, the topic of the message
received from the broker becomes the topic of the message the test
sees.

### Options

This data specifies everything required to attempt the connection
to the MQTT broker.

1. `BrokerURL` (string) is the URL for the MQTT broker.
    
    This required value has the form "PROTOCOL://HOST:PORT".

1. `CertFile` (string) is the optional filename for the client's certificate.

1. `CACertFile` (string) is the optional filename for the certificate
    authority.

1. `KeyFile` (string) is the optional filename for the client's private key.

1. `Insecure` (bool) will given the value for the tls.Config InsecureSkipVerify.
    
    This flag specifies whether a client verifies the server's
    certificate chain and host name. If InsecureSkipVerify is
    true, crypto/tls accepts any certificate presented by the
    server and any host name in that certificate. In this mode,
    TLS is susceptible to machine-in-the-middle attacks unless
    custom verification is used. This should be used only for
    testing.

1. `ALPN` (string) gives the
    https://en.wikipedia.org/wiki/Application-Layer_Protocol_Negotiation
    for the connection.
    
    For example, see
    https://docs.aws.amazon.com/iot/latest/developerguide/protocols.html.

1. `Token` (string) is the optional value for the header given by
    TokenHeader.
    
    See
    https://docs.aws.amazon.com/iot/latest/developerguide/custom-authorizer.html.
    
    When Token is not empty, then you should probably also
    provide AuthorizerName and TokenSig.

1. `TokenHeader` (string) is the name of the header which will have the
    value given by Token.

1. `AuthorizerName` (string) is the optional value for the header
    "x-amz-customauthorizer-name", which is used when making a
    AWS IoT Core WebSocket connection that will call an AWS IoT
    custom authorizer.
    
    See
    https://docs.aws.amazon.com/iot/latest/developerguide/custom-authorizer.html.

1. `TokenSig` (string) is the signature for the token for a WebSocket
    connection to AWS IoT Core.
    
    See
    https://docs.aws.amazon.com/iot/latest/developerguide/custom-authorizer.html.

1. `BufferSize` (int) specifies the capacity of the internal Go
    channel.
    
    The default is DefaultMQTTBufferSize.

1. `PubTimeout` (int64) is the timeout in milliseconds for MQTT PUBACK.

1. `SubTimeout` (int64) is the timeout in milliseconds for MQTT SUBACK.

1. `ClientID` (string) is MQTT client id.

1. `Username` (string) is the optional MQTT client username.

1. `Password` (string) is the optional MQTT client password.

1. `CleanSession` (bool) true, will not resume a previous MQTT
    session for this client id.

1. `WillEnabled` (bool) true, will establish an MQTT Last Will and Testament.
    
    See WillTopic, WillPayload, WillQoS, and WillRetained.

1. `WillTopic` (string) gives the MQTT LW&T topic.
    
    See WillEnabled.

1. `WillPayload` (string) gives the MQTT LW&T payload.
    
    See WillEnabled.

1. `WillQoS` (uint8) specifies the MQTT LW&T QoS.
    
    See WillEnabled.

1. `WillRetained` (bool) specifies the MQTT LW&T retained flag.
    
    See WillEnabled.

1. `KeepAlive` (int64) is the duration in seconds that the MQTT client
    should wait before sending a PING request to the broker.

1. `PingTimeout` (int64) is the duration in seconds that the client will
    wait after sending a PING request to the broker before
    deciding that the connection has been lost.  The default is
    10 seconds.

1. `ConnectTimeout` (int64) is the duration in seconds that the MQTT
    client will wait after attempting to open a connection to
    the broker.  A duration of 0 never times out.  The default
    30 seconds.
    
    This property does not apply to WebSocket connections.

1. `MaxReconnectInterval` (int64) specifies maximum duration in
    seconds between reconnection attempts.

1. `AutoReconnect` (bool) turns on automation reconnection attempts
    when a connection is lost.

1. `WriteTimeout` (int64) is the duration to wait for a PUBACK.

1. `ResumeSubs` (bool) enables resuming of stored (un)subscribe
    messages when connecting but not reconnecting if
    CleanSession is false.

