# secure.helloworld
A version of the grpc Hello World example google.golang.org/grpc/grpc/helloworld 
with OAUTH and TLS

The source code at google.golang.org/grpc gives example of Go applications
communicating via grpc over an http connection without any authentication.
In the real world, a grpc connection should be authenticated or
authorised.  Depending on the mechanism used to do that it may also need
to use an https connection.

This is a reworked version of the hello world example to show how this can be
done.  The client identifies itself using an OAUTH token.  To prevent somebody
intercepting the requests, copying the token and issuing their own bogus
requests, the connection is made through an https channel.

This is work in progress.The TLS connection is correctly done,
but at present the OAUTH token is a hard-wired fake.  The
client always issues the same token, and the server expects to see only that
token.  I plan that in a future version, the client will fetch a token at
run time from an OAUTH framework and the server will use the same framework to
validate the token.

Installation
============

Install the prerequisites.

gRPC uses google's protobuf package so you need to install that.
as well as gRPC itself.
The instructions are [here](https://github.com/golang/protobuf),
but they are incomplete.
You need to install the C++ protocol buffer software as it says,
but you need to run all these "go get" commands:

```
go get golang.org/x/oauth2
go get golang.org/x/net
go get golang.org/x/text
go get cloud.google.com/go
go get -u cloud.google.com/go
go get github.com/golang/protobuf/proto 
go get github.com/golang/protobuf/protoc-gen-go
go get google.golang.org/grpc                   # Google's gRPC examples
go get github.com/goblimey/grpc                 # My rework
```

I think that's all of them.

Now install Google's helloworld example:
```
go install google.golang.org/grpc/examples/greeter_client
go install google.golang.org/grpc/examples/greeter_server
```

and my rework of it:

```
go install github.com/goblimey/grpc/secure_greeter_client
go install github.com/goblimey/grpc/secure_greeter_client
```

Running the examples
====================

(These instructions assume that your Go project bin directory is in your path.
If not, you will need to supply the pathnames of the programs when you run them.)

To run Google's examples,
start two terminal windows.
Run the server in one window:

```
$ greeter_client
```

and the client in the other:

```
$ greeter_client
2017/03/03 08:41:07 Greeting: Hello world

```

To run the secure version of the server you need a certificate file and a key file containing the certificate's private key.
If you already have a digital certificate
from an organisation like
Verisign or Let's Encrypt,
you should be able to use that.
Otherwise, you can generate a self-signed certificate and key using lc-tlscert
as shown below.

If you run both client and server on the same machine,
use a self-signed certificate.

If you have a server with a DNS name,
you can run the greeter server on that and
run the greeter client on your local machine.
The -server option tells the client the name of the server
to connect to.

For that to work,
your server must have a complete set of
Domain Name Service (DNS) records,
including the reverse lookup - the client must be able to 
translate the server name to its IP address and translate 
its IP address back to the domain name.
If there is no reverse DNS translation,
the connection will be slow and unreliable at best
and may not work at all.

This is how you create the self-signed certificate on the server machine:

```
github.com/driskell/log-courier
go install github.com/driskell/log-courier/lc-tlscert
lc-tlscert
```

This will ask a series of questions.
The first is the common name of the cerificate.
use your server name as the common name.
If you run both client and server on the same machine, you can use use the common name 
"localhost", otherwise use the DNS name, 'mydomain.com' or whatever.

To run the server:

```
$ secure_greeter_server -certfile={name of .crt file} -keyfile=\{name of .key file}
```

To run the client on a different machine from the server, you need a
copy of the certificate file on the client machine.
(You don't need to copy the .key file.)  

To run the client:

```
$ secure_greeter_client -server {server name} -certfile={name of .crt file}
```

The -v (verbose) option at both ends shows what goes on behind the scenes.

The -h option shows all the other options.

The client needs to know the name of the server (the -server option).
The default is localhost.

```
$ secure_greeter_client -certfile={name of .crt file}  # connect to localhost
```

The server name MUST match the common name in the certificate.
If you create a certificate with common name mydomain.com, the client
must connect using that name,
even if it's running on the same machine.
If you connect using localhost instead,
you will get this error:

```
2017/03/03 08:47:45 grpc: Server.Serve failed to complete security handshake from "127.0.0.1:42956": read tcp 127.0.0.1:50061->127.0.0.1:42956: read: connection reset by peer
2017/03/03 08:47:45 could not greet: rpc error: code = Internal desc = connection error: desc = "transport: x509: certificate is valid for mydomain.com, not localhost"
```

You can create two certificates, one for mydomain.com and one for localhost.

The software is distributed under the same licence conditions as the original from 
Google.
