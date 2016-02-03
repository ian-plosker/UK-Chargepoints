// Copyright 2014 Orchestrate, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gorc2

import (
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// For older go releases (specifically 1.2 and earlier) there is an issue with
// COMODO certificates since they sign with sha384 which is not included by
// default. As such we force include the package to get the support. On newer
// golang installs this does nothing.
// For more information see this blog post:
//  http://bridge.grumpy-troll.org/2014/05/golang-tls-comodo/
import _ "crypto/sha512"

var (
	// This is the default hostname that will be queried for API calls.
	DefaultAPIHost = "api.cl1.orchestrate.io"

	// The default timeout that will be used for connections. This is used
	// with the default Transport to establish how long a connection attempt
	// can take. This is not the data transfer timeout. Changing this will
	// impact all new connections made with the default transport.
	DefaultDialTimeout = 3 * time.Second

	// This is the default http.Transport that will be associated with new
	// clients. If overwritten then only new clients will be impacted, old
	// clients will continue to use the pre-existing transport.
	DefaultTransport http.RoundTripper = &http.Transport{
		// In the default configuration we allow 4 idle connections to the
		// api server. This limits the number of live connections to our
		// load balancer which reduces load. If needed this can be increased
		// for high volume clients.
		MaxIdleConnsPerHost: 4,

		// This timeout value is how long the http client library will wait
		// for data before abandoning the call. If this is set too low then
		// high work calls, or high latency connections can trip timeouts
		// too often.
		ResponseHeaderTimeout: 3 * time.Second,

		// The default Dial function is over written so it uses net.DialTimeout
		// instead.
		Dial: dialFunc,
	}
)

// A dial function for the DefaultTransport.
func dialFunc(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, DefaultDialTimeout)
}

// We keep the client version here. This is updated when arbitrarily,
// but should change any time we need to track the version that a
// given client is actually using.
const clientVersion = 1

// The user agent that should be sent to the Orchestrate servers.
var userAgent string = fmt.Sprintf("gorc2/%d (%s)",
	clientVersion, runtime.Version())

// This user agent is used if a client has been using deprecated functions.
// Then intention is to allow us reach out to the users prior to removing
// them from the client.
var userAgentDeprecated string = fmt.Sprintf("gorc2/%d (%s) [deprecated]",
	clientVersion, runtime.Version())

//
// Client
//

// An Orchestrate Client object.
type Client struct {
	// This is the host name that will be used in client queries. By default
	// this will be set to DefaultAPIHost, and if this is left empty
	// then that default will be used as well.
	APIHost string

	// This is the HTTP client that will be used to perform HTTP queries
	// against Orchestrate.
	HTTPClient *http.Client

	// The authorization token passed into NewClient().
	authToken string

	// This value will be automatically set to a non zero value if a call is
	// made to any deprecated function.
	deprecated int32
}

// Returns a new Client object that will use the given authToken for
// authorization against Orchestrate. This token can be obtained
// at https://dashboard.orchestrate.io
func NewClient(authToken string) *Client {
	return &Client{
		APIHost:    DefaultAPIHost,
		HTTPClient: nil,
		authToken:  authToken,
	}
}

// Returns a Collection object for a collection with the given name. Note that
// this call does not verify that the collection exists.
func (c *Client) Collection(name string) *Collection {
	return &Collection{
		client: c,
		Name:   name,
	}
}

// Check that Orchestrate is reachable.
func (c *Client) Ping() error {
	//	return nil
	_, err := c.emptyReply("HEAD", "", nil, nil, 200)
	return err
}

// Executes an HTTP request.
func (c *Client) doRequest(
	method, trailing string, headers map[string]string, body io.Reader,
) (*http.Response, error) {
	// Get the URL that we should be talking too.
	host := c.APIHost
	if host == "" {
		host = DefaultAPIHost
	}
	url := "http://" + host + "/v0/" + trailing

	// Create the new Request.
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Ensure that the query gets the authToken as username.
	req.SetBasicAuth(c.authToken, "")

	// Add any headers that the client provided.
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	if atomic.LoadInt32(&c.deprecated) == 0 {
		req.Header.Add("User-Agent", userAgent)
	} else {
		req.Header.Add("User-Agent", userAgentDeprecated)
	}

	// If the client request has a body then we need to set a Content-Type
	// header.
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	// If the HTTPClient is nil we use the DefaultTransport provided in this
	// package, otherwise we use the specific HTTPClient that the caller set
	// in the client object.
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Transport: DefaultTransport}
	}
	return client.Do(req)
}

// This call will perform a simple request which expects no body to be
// returned. These are typically sued with POST/PUT/DELETE type calls which
// expect no response from the server.
//
// Any status return other than 'status' will cause an error to be returned
// from this function.
func (c *Client) emptyReply(
	method, path string, headers map[string]string, body io.Reader, status int,
) (*http.Response, error) {
	resp, err := c.doRequest(method, path, headers, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check the status code.
	if resp.StatusCode != status {
		return nil, newError(resp)
	}

	// Read the whole body to ensure that the connections can be reused. Note
	// that we don't bother checking errors here since an error will not impact
	// the code path at all.
	io.Copy(ioutil.Discard, resp.Body)

	// Success!
	return resp, nil
}

// This call will perform a request which expects a JSON body to be returned.
// The contents of the body will be decoded into the value given.
//
// Any status return other than 'status' will cause an error to be returned
// from this function.
func (c *Client) jsonReply(
	method, path string, body io.Reader, status int, value interface{},
) (*http.Response, error) {
	headers := map[string]string{"Accept-Encoding": "gzip; deflate"}
	resp, err := c.doRequest(method, path, headers, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Ensure that the returned status was expected.
	if resp.StatusCode != status {
		return nil, newError(resp)
	}

	// See what kind of encoding the server is replying with.
	var decoder *json.Decoder
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		decoder = json.NewDecoder(gzipReader)
	case "deflate":
		decoder = json.NewDecoder(flate.NewReader(resp.Body))
	default:
		decoder = json.NewDecoder(resp.Body)
	}


	// Decode the body into a json object.
	if err := decoder.Decode(value); err != nil {
		return nil, err
	}

	// Success!
	return resp, nil
}
