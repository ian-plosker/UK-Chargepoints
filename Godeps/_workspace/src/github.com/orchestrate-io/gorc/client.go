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

// A client for use with Orchestrate.io: http://orchestrate.io/
//
// Orchestrate unifies multiple databases through one simple REST API.
// Orchestrate runs as a service and supports queries like full-text
// search, events, graph, and key/value.
//
// You can sign up for an Orchestrate account here:
// http://dashboard.orchestrate.io
package gorc

import (
	"encoding/json"
	"errors"
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
	DefaultAPIHost = "api.orchestrate.io"

	// The default timeout that will be used for connections. This is used
	// with the default Transport to establish how long a connection attempt
	// can take. This is not the data transfer timeout. Changing this will
	// impact all new connections made with the default transport.
	DefaultDialTimeout = 3 * time.Second

	// This is the default http.Transport that will be associated with new
	// clients. If overwritten then only new clients will be impacted, old
	// clients will continue to use the pre-existing transport.
	DefaultTransport *http.Transport = &http.Transport{
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
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, DefaultDialTimeout)
		},
	}
)

// We keep the client version here. This is updated when arbitrarily,
// but should change any time we need to track the version that a
// given client is actually using.
const clientVersion = 3

// The user agent that should be sent to the Orchestrate servers.
var userAgent string = fmt.Sprintf("gorc/%d (%s)",
	clientVersion, runtime.Version())

// This user agent is used if a client has been using deprecated functions.
// Then intention is to allow us reach out to the users prior to removing
// them from the client.
var userAgentDeprecated string = fmt.Sprintf("gorc/%d (%s) [deprecated]",
	clientVersion, runtime.Version())

// A representation of a Key/Value object's path within Orchestrate.
type Path struct {
	Collection string `json:"collection"`
	Key        string `json:"key"`
	Ref        string `json:"ref"`
}

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
// at http://dashboard.orchestrate.io
func NewClient(authToken string) *Client {
	return &Client{
		APIHost:    DefaultAPIHost,
		HTTPClient: nil,
		authToken:  authToken,
	}
}

// This function is deprecated. Please just set the HTTPClient field on
// the client object manually.
func NewClientWithTransport(
	deprecated_authToken string, deprecated_transport *http.Transport,
) *Client {
	client := NewClient(deprecated_authToken)
	client.HTTPClient = &http.Client{Transport: deprecated_transport}
	client.deprecated = 1
	return client
}

// Check that Orchestrate is reachable.
func (c *Client) Ping() error {
	resp, err := c.doRequest("HEAD", "", nil, nil)
	if err != nil {
		return err
	}

	// If the request ended in error then read the body into an
	// OrchestrateError object.
	if resp.StatusCode != 200 {
		return newError(resp)
	}

	// Read the body so the connection can be properly reused.
	io.Copy(ioutil.Discard, resp.Body)

	return nil
}

// Executes an HTTP request.
func (c *Client) doRequest(method, trailing string, headers map[string]string, body io.Reader) (*http.Response, error) {
	// Get the URL that we should be talking too.
	host := c.APIHost
	if host == "" {
		host = DefaultAPIHost
	}
	url := "https://" + host + "/v0/" + trailing

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

//
// OrchestrateError
//

// An implementation of 'error' that exposes all the orchestrate specific
// error details.
type OrchestrateError struct {
	// The status string returned from the HTTP call.
	Status string `json:"-"`

	// The status, as an integer, returned from the HTTP call.
	StatusCode int `json:"-"`

	// The Orchestrate specific message representing the error.
	Message string `json:"message"`
}

// Creates a new OrchestrateError from a given http.Response object.
func newError(resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	oe := &OrchestrateError{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}
	if err := json.Unmarshal(body, oe); err != nil {
		return errors.New(string(body))
	}

	return oe
}

// Convert the error to a meaningful string.
func (e OrchestrateError) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.Status, e.StatusCode, e.Message)
}
