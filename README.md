# goff [![GoDoc](https://godoc.org/github.com/Forestmb/goff?status.png)](https://godoc.org/github.com/Forestmb/goff) [![Build Status](https://travis-ci.org/Forestmb/goff.png?branch=master)](https://travis-ci.org/Forestmb/goff) [![Coverage Status](https://img.shields.io/coveralls/Forestmb/goff.svg)](https://coveralls.io/r/Forestmb/goff) #

goff is a library for communicating with the [Yahoo Fantasy Sports APIs](
http://developer.yahoo.com/fantasysports/guide/).

This application is written using the Go programming language and is licensed
under the [New BSD license](
https://github.com/Forestmb/goff/blob/master/LICENSE).

## Building ##

    $ go get https://github.com/Forestmb/goff
    $ cd $GOPATH/src/github.com/Forestmb/goff
    $ ./build.sh

To make sure this build runs before every commit, use:

    $ ln -s "$(pwd)/build.sh" .git/hooks/pre-commit

## Debug ##

The `goff/debug` package can be used to help with developing `goff`. It uses
the OAuth 2.0 configuraiton provided by `goff` to make arbitrary `GET` request
to the Yahoo Fantasy Sports APIs and outputs the string XML response. To run:

    $ cd $GOPATH/src/github.com/Forestmb/goff
    $ go run debug/debug.go --clientKey=<key> --clientSecret=<secret> --redirectURL=<redirect-url>

The values `key` and `secret` can be obtained after registering your own
applicaiton: http://developer.yahoo.com/fantasysports/guide/GettingStarted.html

The `redirect-url` must match the callback domain given when registering the
application with Yahoo. For development, using `127.0.0.1` for the domain and
`http://127.0.0.1` as the redirect URL should suffice.
