# secure.helloworld
A version of the grpc Hello World example google.golang.org/grpc/grpc/helloworld 
with TLS

The source code at google.golang.org/grpc gives examples of Go applications
communicating via grpc over a clear-text http connection.
Many applications require an encrypted HTTPS connection.
The example here shows how to do that.
It's a reworked version of Google's hello world example.

The solution is in two parts,
a client and a server.
The client identifies itself using an authorization token.
This version uses a fixed token.
In the real world, the token would be different eaxh time,
created and validated by a system such as OAUTH.


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

You can run the secure client and server on your local machine
in the same way,
but they need a bit more information.

A TLS connection needs a matching pair of files containing a certificate and a key.
The key is a private key and the certificate contains the matching public key.
For a self-contained system like this
where you control both the client and server software,
you can create a self-signed certificate using the lc-tlscert application:

```
github.com/driskell/log-courier
go install github.com/driskell/log-courier/lc-tlscert
lc-tlscert
```

This will ask a series of questions which it uses to fill in the certificate.
The first is the common name of the certificate - use localhost.
The rest of the questions ask for things like your address.
You can give dummy values for those.

Now you can run the secure server:

```
$ secure_greeter_server -certfile={name of crt file} -keyfile={name of .key file}
```

and the secure client:

```
$ secure_greeter_client -certfile={name of .crt file}  # connect to localhost
2017/03/04 18:15:10 Greeting: Hello world
```

That test is a bit artificial.
In a real application
the client and server will usually run on different machines.
To test that the solution works across an Internet connection, 
I ran the server on a Virtual Private Server (VPS),
a Digital Ocean "droplet"
with a fixed IP address and a domain name.
For a client machine I used a Pine64 single board computer.
The client ran at my house connecting
through a wired connection to a fairly fast cable modem.

For this sort of test,
the server is run in the same way as before.
The client has to know the server's name so it's run like so:
```
$ secure_greeter_client -server={mydomain.com} -certfile={.crt file}
```

The -server option tells the client the name of the server
to connect to.  The default is localhost.
You can also set the port if you need to 
using the -p option.
The default is 50061.

The results of this test were mixed.
The client worked on most of the attempts,
but in some cases only after some error messages appeared
showing that it was retrying the request.
A few attempts failed altogether,
failing to contact the server.
I think some tuning of the client and server is required,
perhaps increasing the request timeout period and 
the number of retries.
 
To run the secure greeter server on a remote machine like this,
you need to create the certificate on the server machine 
and set the common name to its DNS name.
Then you need to copy the certificate file to your client machine.
If you have a digital certificate for your server
from an organisation like
Verisign or Let's Encrypt,
you should be able to use that 
rather than creating a self-signed certificate.

To support a TLS connection from a remote client,
your server must have a complete set of
Domain Name Service (DNS) records,
including the reverse translation - the client must be able to 
translate the server's name to its IP address and translate 
its IP address back to it's domain name.
If there is no reverse DNS translation,
the connection will be slow and unreliable at best
and may not work at all.
(In the case of a Digital Ocean droplet,
you set the droplet name to be the domain name
and a reverse DNS record is automatically created.)

The server name that you give to the client when you run it
MUST match the common name in the certificate.
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

Licence
=========
This software is distributed under the same licence conditions as Google's original.
See the source code.
