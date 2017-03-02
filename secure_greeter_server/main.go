/*
 * The source code at google.golang.org/grpc/ gives example of Go applications
 * communicating via grpc over an http connection without any authentication.
 * In the real world, a grpc connection should be authenticated or
 * authorised.  Depending on the mechanism used to do that, it may also need
 * to use an https connection to prevent a "man in the middle" attack.
 *
 * This is a reworked version of the hello world example to show how this can be
 * done.  The client identifies itself using an OAUTH token.  To prevent somebody
 * intercepting the requests, copying the token and issuing their own bogus
 * requests, the connection is made through an https channel.
 *
 * This is work in progress.  At present the OAUTH token is a hard-wired fake.  The
 * client always issues the same token, and the server expects to see only that
 * token.  I plan that in a future version, the client will fetch a token at run
 * time from an OAUTH framework and the server will use the same framework to
 * validate the token.
 *
 * Simple usage:
 *
 *     $ secure_greeter_server \
 *         --certfile=/home/simon/ca.certificate/selfsigned.crt \
 *         --keyfile=/home/simon/ca.certificate/selfsigned.key
 *
 * This software is Copyright 2015 Google and 2017 Simon Ritchie.  It's distributed
 * under the same licence conditions as the original from Google:
 *
 * Copyright 2015, Google Inc.
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *     * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *     * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 *     * Neither the name of Google Inc. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	pb "github.com/goblimey/grpc/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpccred "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

var (
	verbose  = flag.Bool("v", false, "verbose mode")
	port     = flag.Int("p", 50061, "port")
	certfile = flag.String("certfile", "", "certificate file")
	keyfile  = flag.String("keyfile", "", "private key file")
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {

	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	flag.Parse()

	portStr := ":" + strconv.Itoa(*port) // ":50061"
	lis, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// The server options control the style of the gRPC connection, for example
	// encrypted (https) or plain text (http).
	var opts []grpc.ServerOption

	// Create a server option from the OAUTH interceptor.
	opts = append(opts, grpc.UnaryInterceptor(OAuthUnaryInterceptor))

	// Creating a server option for the TLS connaction is more complicated.  The
	// setup uses wisdom from:
	//
	//     http://stackoverflow.com/questions/22666163/golang-tls-with-selfsigned-certificate
	// and
	//    http://www.bite-code.com/2015/06/25/tls-mutual-auth-in-golang/
	//
	// To make the connection work you need a self-signed certificate and a
	// matching private key.  Create these using lc-tlscert:
	//
	//    go get github.com/driskell/log-courier
	//    go install github.com/driskell/log-courier/lc-tlscert
	//    lc-tlscert
	//    (Give your server name as the common name)
	//
	// The common name must match the server name that the client will use to
	// connect.  If the client and server are on the same machine you can use
	// "localhost".
	//
	// lc-tlscert produces a .key file containing your server's private key and a
	// .cert file containing a certificate.  The client needs to be able to see the
	// certificate data.  If the client is on another machine, you need to provide
	// it with a copy of the .crt file.
	//
	// Danger Will Robinson:  I found instructions on the web showing other ways to
	// generate a self-signed certificate but the result didn't work for gRPC.
	//
	// Load509KeyPair doesn't return an error if the files don't exist(!) so we
	// check that they do before trying to use them.

	if len(*certfile) == 0 || len(*keyfile) == 0 {
		log.Fatalf("you must specify the cert file and the key file")
	}

	if _, err := os.Stat(*keyfile); os.IsNotExist(err) {
		log.Fatalf("cannot open the key file %s", *keyfile)
	}

	if _, err := os.Stat(*certfile); os.IsNotExist(err) {
		log.Fatalf("cannot open the cert file %s", *certfile)
	}

	// Load the public certificate and the private key files.
	cert, err := tls.LoadX509KeyPair(*certfile, *keyfile)

	config := tls.Config{Certificates: []tls.Certificate{cert}}

	// Create the TLS server option.
	serverOption := grpc.Creds(grpccred.NewTLS(&config))

	// Create the gRPC server.
	opts = append(opts, serverOption)

	s := grpc.NewServer(opts...)

	// Register the server.
	pb.RegisterGreeterServer(s, &server{})

	// Register the reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// OAuthUnaryInterceptor intercepts the gRPC request, extracts the OAUTH token and
// the user-id and validates them.  This version uses the wisdom in
//
//     https://godoc.org/google.golang.org/grpc#UnaryServerInterceptor
// and
//     https://texlution.com/post/oauth-and-grpc-go/
func OAuthUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	// retrieve metadata from context
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "no metadata in context")
	}

	// validate the 'authorization' metadata
	// like headers, the value is an slice []string
	uid, err := validateOAUTHToken(md["authorization"])
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "authentication failed - %s",
			err.Error())
	}

	// add the user ID to the context
	newCtx := context.WithValue(ctx, "user_id", uid)

	// handle scopes?
	// ...
	return handler(newCtx, req)
}

// validateOAUTHToken searches through a slice of authorization headers.  If it
// finds any containing an OAUTH token it validates them.  It reurns the ID of the
// user that owns the first valid token that it finds.
//
// This version is a fake.  It has a hard-wired OAUTH token.  It accepts only that
// and if it finds it, return userID 2.  In a real application it would use an
// OAUTH server to validate and fetch the user ID.
func validateOAUTHToken(authHeaders []string) (uint64, error) {
	if *verbose {
		log.Printf("%d authorization headers", len(authHeaders))
	}
	for i := range authHeaders {
		if *verbose {
			if *verbose {
				log.Printf("authorization header %s", authHeaders[i])
			}
			if authHeaders[i] == "Bearer rTO69tZATgSqamjQn7v9HA" {
				if *verbose {
					log.Printf("authorised")
				}
				return 2, nil
			}
		}
	}

	// no valid auth header found
	if *verbose {
		log.Printf("authorisation failed")
	}
	return 0, errors.New("no valid authorization header")

}
