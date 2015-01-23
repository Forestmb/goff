// Command debug starts a terminal based fantasy client using the OAuth
// consumer provided by package goff.
//
//     Usage: go run debug/debug.go --clientKey=<key> --clientSecret=<secret>
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/Forestmb/goff"
)

func main() {
	clientKey := flag.String(
		"clientKey",
		"",
		"Required client OAuth key. "+
			"See http://developer.yahoo.com/fantasysports/guide/GettingStarted.html"+
			" for more information")
	clientSecret := flag.String(
		"clientSecret",
		"",
		"Required client OAuth secret. "+
			"See http://developer.yahoo.com/fantasysports/guide/GettingStarted.html"+
			" for more information")
	flag.Parse()
	if len(*clientKey) == 0 || len(*clientSecret) == 0 {
		fmt.Println("Usage: debug --clientKey=\"<key>\" --clientSecret=\"<secret>\"")
		os.Exit(1)
	}

	fmt.Fprintf(
		os.Stdout,
		"clientKey: %s, clientSecret: %s\n",
		*clientKey,
		*clientSecret)

	consumer := goff.GetConsumer(*clientKey, *clientSecret)

	requestToken, url, err := consumer.GetRequestTokenAndUrl("oob")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting request token: %s", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "(1) Go to: "+url)
	fmt.Fprintln(os.Stdout, "(2) Grant access, you should get back a verification code.")
	fmt.Fprint(os.Stdout, "(3) Enter that verification code here: ")

	verificationCode := ""
	fmt.Scanln(&verificationCode)

	accessToken, err := consumer.AuthorizeToken(requestToken, verificationCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error authorizing token: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "Access granted")
	fmt.Fprintln(
		os.Stdout,
		"See https://developer.yahoo.com/fantasysports/guide/ResourcesAndCollections.html")
	fmt.Fprintln(os.Stdout, "for information about the types of requests available")
	fmt.Fprintln(os.Stdout, "Type 'exit' to quit anytime")
	for {
		fmt.Fprint(os.Stdout, "Enter URL: ")
		url := ""
		fmt.Scanln(&url)
		if url == "exit" || url == "" {
			break
		}

		start := time.Now()
		response, err := consumer.Get(url, map[string]string{}, accessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting content: %s\n", err)
		} else {
			defer response.Body.Close()
			bits, err := ioutil.ReadAll(response.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading content: %s\n", err)
			} else {
				str := string(bits[:])
				fmt.Fprintf(os.Stdout, "Response:\n%s\n", str)
			}
		}
		fmt.Fprintf(os.Stdout, "Request time: %s\n\n", time.Since(start))
	}
}
