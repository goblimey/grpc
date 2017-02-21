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
	"flag"
	"io/ioutil"
	"log"
	"os"

	pb "github.com/goblimey/secure.helloworld/helloworld"
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
	address     = "localhost:50051"
	defaultName = "world"
)

var (
	verbose = flag.Bool("v", false, "verbose mode")
)

func main() {
	flag.Parse()
	// Get OAUTH token.  In the real world the client would get a
	// token from an OAUTH source such as Hydra, and the server would check with the
	// OAUTH source that the token is valid.
	//
	// Currently the token is a hard-wired fake.  The client always sends and the
	// server always expects this value.
	if *verbose {
		log.Printf("getting auth token")
	}
	tokenText := "{\"access_token\":\"rTO69tZATSgSqamjQn7v9HA\",\"expires_in\":3600,\"refresh_token\":\"xBqf2OWbT_KvWW8LHOPF0A\",\"scope\":\"everything\",\"token_type\":\"Bearer\"}"
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenText), &token); err != nil {
		log.Fatalf("error unmarshalling JSON from OAUTH token: %v", err)
	}
	if *verbose {
		log.Printf("got auth token %s type %s", token.AccessToken, token.TokenType)
	}

	// Create the OAUTH dial option from tye token
	credentials := oauth.NewOauthAccess(&token)
	oauthDialOption := grpc.WithPerRPCCredentials(credentials)

	// Load the self-signed CA certificate.  I generated this using Jason Woods'
	// lc_tlscert app, which is part of github.com/driskell/log-courier.  BEWARE
	// I found other instructions on the web to generate a certificate and the
	// result didn't work for this purpose.  Install log-courier and use that:
	//
	//    go get github.com/driskell/log-courier
	//    go intall github.com/driskell/log-courier/lc-tlscert
	//    lc-tlscert
	//
	// Give your server name as the common name (for example localhost)

	caCert, err := ioutil.ReadFile("/home/simon/ca.certificate/goblimey.com.selfsigned.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := tls.Config{RootCAs: caCertPool}

	tlsDialOption := grpc.WithTransportCredentials(grpccred.NewTLS(&tlsConfig))

	if *verbose {
		log.Printf("connecting to server %s", address)
	}
	conn, err := grpc.Dial(address, oauthDialOption, tlsDialOption)
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
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
