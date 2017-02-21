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
 * token.  It is planned that in a future version, the client will fetch a token at
 * run time from an OAUTH framework and the server will use the same framework to
 * validate the token.
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

	pb "github.com/goblimey/secure.helloworld/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpccred "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":50051"
)

var (
	verbose = flag.Bool("v", false, "verbose mode")
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {

	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	// add the interceptor as a server option
	opts = append(opts, grpc.UnaryInterceptor(AuthUnaryInterceptor))

	// TLS connection setup uses a combination of:
	//     http://stackoverflow.com/questions/22666163/golang-tls-with-selfsigned-certificate
	// and
	//    http://www.bite-code.com/2015/06/25/tls-mutual-auth-in-golang/
	//
	// Create the self-signed cert using lc-tlscert:
	//    go get github.com/driskell/log-courier
	//    go install github.com/driskell/log-courier/lc-tlscert

	cert, err := tls.LoadX509KeyPair("/home/simon/ca.certificate/selfsigned.crt",
		"/home/simon/ca.certificate/selfsigned.key")
	config := tls.Config{Certificates: []tls.Certificate{cert}}

	serverOption := grpc.Creds(grpccred.NewTLS(&config))

	opts = append(opts, serverOption)

	s := grpc.NewServer(opts...)

	pb.RegisterGreeterServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// AuthUnaryInterceptor is an interceptor function.  It intercepts the gRPC
// request, extracts the OAUTH token and the user-id and validates them.
// https://godoc.org/google.golang.org/grpc#UnaryServerInterceptor
// https://texlution.com/post/oauth-and-grpc-go/
func AuthUnaryInterceptor(
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

	// validate 'authorization' metadata
	// like headers, the value is an slice []string
	uid, err := ValidationOAUTHToken(md["authorization"])
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "authentication failed - %s",
			err.Error())
	}

	// add user ID to the context
	newCtx := context.WithValue(ctx, "user_id", uid)

	// handle scopes?
	// ...
	return handler(newCtx, req)
}

func ValidationOAUTHToken(authHeaders []string) (uint64, error) {
	if *verbose {
		log.Printf("%d authorization headers", len(authHeaders))
	}
	for i := range authHeaders {
		if *verbose {
			if *verbose {
				log.Printf("authorization header %s", authHeaders[i])
			}
			if authHeaders[i] == "Bearer rTO69tZATSgSqamjQn7v9HA" {
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
