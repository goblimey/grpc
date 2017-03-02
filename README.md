# secure.helloworld
A version of the grpc Hello World example (google.golang.org/grpc/grpc/helloworld) 
with OAUTH and TLS

The source code at google.golang.org/grpc/ gives example of Go applications
communicating via grpc over an http connection without any authentication.
In the real world, a grpc connection should be authenticated or
authorised.  Depending on the mechanism used to do that it may also need
to use an https connection.

This is a reworked version of the hello world example to show how this can be
done.  The client identifies itself using an OAUTH token.  To prevent somebody
intercepting the requests, copying the token and issuing their own bogus
requests, the connection is made through an https channel.

This is work in progress.  At present the OAUTH token is a hard-wired fake.  The
client always issues the same token, and the server expects to see only that
token.  I plan that in a future version, the client will fetch a token at
run time from an OAUTH framework and the server will use the same framework to
validate the token.

The software is distributed under the same licence conditions as the original from Google.



