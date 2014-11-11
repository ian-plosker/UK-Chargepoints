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
	"flag"
	"fmt"
	"net/http"
	"sync"

	"github.com/liquidgecka/testlib"
	"github.com/orchestrate-io/dvr"
)

// This is the auth token  that will be used for queries against Orchestrate.
// If the dvr library is in recording or pass through mode then this is
// the real token that will be used against Orchestrate. If it is in
// replay mode then this will be a fake, obfuscated token that exposes
// no real security credentials.
var testAuthToken string

// This is the obviously obfuscated token that is used when the dvr library
// is in recording mode.
var ObfuscatedAuthToken = "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"

func init() {
	// We need to ensure that all of our testing is done with the dvr library
	// firmly in place. This allows us to quickly replay existing queries
	// rather than constantly having to go back to Orchestrate in testing.
	//
	// This also allows contributors to submit pull requests and such without
	// having to expose real auth tokens..

	// First put the Obfuscator in place.
	dvr.Obfuscator = dvr.BasicAuthObfuscator(ObfuscatedAuthToken, "")

	// Ensure that replay mode is default in the dvr Library.
	dvr.DefaultReplay = true

	// Next inject a dvr.RoundTripper into the DefaultTransport so that all
	// testing queries default to using the RoundTripper.
	DefaultTransport = dvr.NewRoundTripper(DefaultTransport)

	// Insert flags that allow the user to specify an auth token as well as
	// a collection name.
	flag.StringVar(&testAuthToken, "auth_token", "",
		"The Orchestrate auth token for a given application.")
}

// This call will create a client and ensure that all the arguments needed
// have been passed.
func cleanTestingClient(T *testlib.T) *Client {
	if !dvr.IsReplay() {
		if testAuthToken == "" {
			T.Fatalf("" +
				"In order to run the gorc tests you must provide a\n" +
				"authorization token. A new auth token can be obtained via\n" +
				"the dashboard (https://dashboard.orchestrate.io) and\n"+
				"supplied via the -auth_token= flag. Without this flag\n" +
				"this test can not continue.")
		}
		return NewClient(testAuthToken)
	}
	return NewClient(ObfuscatedAuthToken)
}

// Simple type that allows for locking around an error object.
type Locker struct {
	Err error
	sync.Mutex
	sync.WaitGroup
}

// Sets the Err value if its non nil.
func (l *Locker) Set(err error) {
	if err == nil {
		return
	}
	l.Lock()
	l.Err = err
	l.Unlock()
}

// Returns true if Err is non nil.
func (l *Locker) IsSet() (b bool) {
	l.Lock()
	b = l.Err != nil
	l.Unlock()
	return
}

// Removes all items from the test collection.
func PurgeTheTestCollection(T *testlib.T, collection *Collection) {
	locker := Locker{}
	in := make(chan *Item, 100)
	for i := 0; i < 25; i++ {
		locker.Add(1)
		go func() {
			defer locker.Done()
			for item := range in {
				if locker.IsSet() {
					continue
				}
				locker.Set(collection.Purge(item.Key))
			}
		}()
	}

	// Get a listing of all the items that are currently in the collection.
	lister := collection.List(&ListQuery{Limit: 100})
	x := 0
	for lister.Next() {
		x++
		item, err := lister.Get(nil)
		T.ExpectSuccess(err)
		in <- item
	}
	T.ExpectSuccess(lister.Error)
	close(in)

	// Wait for the purgers.
	locker.Wait()
	T.ExpectSuccess(locker.Err)
}

// This type can not be marshaled. It is used for code coverage testing.
type jsonMarshalError struct {
}

func (j *jsonMarshalError) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("JSON MARSHAL ERROR")
}

// This type can not be unmarshaled. It is used for code coverage testing.
type jsonUnmarshalError struct {
}

func (j *jsonUnmarshalError) UnmarshalJSON(data []byte) error {
	return fmt.Errorf("JSON UNMARSHAL ERROR")
}

// Runs a function on the results returned from the server prior to it being
// returned back into the client.
type testRoundTripper struct {
	f            func(*http.Response)
	RoundTripper http.RoundTripper
}

func (t *testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(r)
	t.f(resp)
	return resp, err
}
