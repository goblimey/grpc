/*
 * The source code at google.golang.org/grpc/ gives example of Go applications
 * communicating via grpc over an http connection without any authentication.
 * In the real world, a grpc connection should be authenticated or
 * authorised.  Depending on the mechanism used to do that, it may also need
 * to use an https connection to prevent a "man in the middle" attack.
 *
 * This is a version of the client from the hello world example, reworked to show
 * how this can be done.  The client identifies itself using an OAUTH token.  To
 * prevent somebody intercepting the requests, copying the token and issuing their
 * own bogus requests, the connection is made through an https channel.
 *
 * This is work in progress.  At present the OAUTH token is a hard-wired fake.  The
 * client always issues the same token, and the server expects to see only that
 * token.  I plan that in a future version, the client will fetch a token at run
 * time from an OAUTH framework and the server will use the same framework to
 * validate the token.
 *
 * Simple usage (localhost):
 *
 *    $ secure_greeter_client \
 *         -certfile=/home/simon/ca.certificate/selfsigned.crt
 *
 * The original software is Copyright 2015 Google and the changes 2017 Simon
 * Ritchie.  This version is distributed under the same licence conditions as
 * the original from Google:
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
	"flag"
	"io/ioutil"
	"log"
	"strconv"

	pb "github.com/goblimey/grpc/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"crypto/tls"
	"crypto/x509"
	"encoding/json"

	"golang.org/x/oauth2"
	grpccred "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

const (
	defaultName = "world"
)

var (
	verbose  = flag.Bool("v", false, "verbose mode")
	port     = flag.Int("p", 50061, "port")
	server   = flag.String("server", "localhost", "the server")
	certfile = flag.String("certfile", "", "the certificate file")
)

func main() {
	flag.Parse()

	address := *server + ":" + strconv.Itoa(*port) // "localhost;50061"

	// The dial options control the style of connection, for example encrypted
	// (https) or plain text (http).
	var opts []grpc.DialOption

	// Get an OAUTH token and create an OAUTH dial option.
	//
	// Currently the token is a hard-wired fake.  The client always sends this
	// token and the server always expects to receive it.  In the real world the
	// client would get a token from an OAUTH source such as a Hydra system, and
	// the server would check with the OAUTH server that the token is valid.
	if *verbose {
		log.Printf("getting auth token")
	}
	tokenText := "{\"access_token\":\"rTO69tZATgSqamjQn7v9HA\",\"expires_in\":3600,\"refresh_token\":\"xBqf2OWbT_KvWW8LHOPF0A\",\"scope\":\"everything\",\"token_type\":\"Bearer\"}"
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenText), &token); err != nil {
		log.Fatalf("error unmarshalling JSON from OAUTH token: %v", err)
	}
	if *verbose {
		log.Printf("got auth token %s type %s", token.AccessToken, token.TokenType)
	}

	// Create the OAUTH dial option from the token
	credentials := oauth.NewOauthAccess(&token)
	oauthDialOption := grpc.WithPerRPCCredentials(credentials)

	// add the interceptor as a server option
	opts = append(opts, oauthDialOption)

	// Load the self-signed CA certificate.  If the client and server run on
	// different machines you have to generate this on the server and copy it
	// to the client machine.  I generated mine using Jason Woods' lc_tlscert app:
	//
	//    go get github.com/driskell/log-courier
	//    go intall github.com/driskell/log-courier/lc-tlscert
	//    lc-tlscert
	//    (Give your server name as the common name)
	//
	// The common name must match the server name that you use when you run the
	// client.  If the client and server are on the same machine you can use
	// "localhost".
	//
	// That processes creates a .crt file and a .key file.  You only need a copy of
	// the .crt file.  The .key file contains the private key and it stays on the
	// server.
	//
	// Danger Will Robinson:  I found instructions on the web showing other ways to
	// generate a self-signed certificate but the result didn't work for gRPC.

	caCert, err := ioutil.ReadFile(*certfile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := tls.Config{RootCAs: caCertPool}

	tlsDialOption := grpc.WithTransportCredentials(grpccred.NewTLS(&tlsConfig))
	// add the TLS as a server option
	opts = append(opts, tlsDialOption)

	if *verbose {
		log.Printf("connecting to server %s", address)
	}
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	if *verbose {
		log.Printf("connected to server")
	}

	// Set up a connection to the server.
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(flag.Args()) > 1 {
		name = flag.Arg(1)
	}
	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
