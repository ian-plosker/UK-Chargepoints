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
	"net/http"
	"testing"
	"time"

	"github.com/liquidgecka/testlib"
)

func TestCollection_AddEvent(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// We use this value as the payload to the event.
	key := "addevent"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Update(key, value)
	T.ExpectSuccess(err)

	// Next we attempt to add an event to the item.
	event, err := collection.AddEvent(key, "type1", value)
	T.ExpectSuccess(err)

	// All is good. Now attempt to get the event to ensure that it exists.
	var valueGet map[string]interface{}
	eventGet, err := collection.GetEvent(key, "type1",
		event.Timestamp, event.Ordinal, &valueGet)
	T.ExpectSuccess(err)
	T.Equal(valueGet, value)
	T.Equal(eventGet, event)
}

func TestCollection_AddEventWithTimestamp(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// We use this value as the payload to the event.
	key := "addeventwithtimestamp"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Update(key, value)
	T.ExpectSuccess(err)

	// Next we attempt to add an event to the item.
	event, err := collection.AddEventWithTimestamp(key, "type1",
		time.Unix(100000, 100), value)
	T.ExpectSuccess(err)

	// All is good. Now attempt to get the event to ensure that it exists.
	var valueGet map[string]interface{}
	eventGet, err := collection.GetEvent(key, "type1", event.Timestamp,
		event.Ordinal, &valueGet)
	T.ExpectSuccess(err)
	T.Equal(valueGet, value)
	T.Equal(eventGet, event)
}

func TestCollection_innerAddEvent(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// We use this value as the payload to the event.
	key := "inner_addevent"
	keyFailure := "inner_addevent_failure"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Update(key, value)
	T.ExpectSuccess(err)
	_, err = collection.Update(keyFailure, value)
	T.ExpectSuccess(err)

	// Failure Condition 1: json Marshal error.
	func() {
		event, err := collection.innerAddEvent(keyFailure, "type1", nil,
			&jsonMarshalError{})
		T.ExpectErrorMessage(err, "JSON MARSHAL ERROR")
		T.Equal(event, nil)

	}()

	// Failure condition 2: client error.
	func() {
		defer func(s string) { client.APIHost = s }(client.APIHost)
		client.APIHost = "localhost:100000"
		event, err := collection.innerAddEvent(keyFailure, "type1", nil, value)
		T.ExpectErrorMessage(err, "dial tcp: invalid port 100000")
		T.Equal(event, nil)
	}()

	// Failure Condition 3: Missing Location header.
	func() {
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		event, err := collection.innerAddEvent(keyFailure, "type1", nil, value)
		T.ExpectErrorMessage(err, "Missing Location header.")
		T.Equal(event, nil)
	}()

	// Failure Condition 4: Malformed location header.
	func() {
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
					resp.Header.Add("Location", "XXX")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		event, err := collection.innerAddEvent(keyFailure, "type1", nil, value)
		T.ExpectErrorMessage(err, "Malformed Location header.")
		T.Equal(event, nil)
	}()

	// Failure Condition 5: Malformed ordinal format.
	func() {
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
					resp.Header.Add("Location",
						"/v0/collection/key/events/type1/100/XXX")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		event, err := collection.innerAddEvent(keyFailure, "type1", nil, value)
		T.ExpectErrorMessage(err, "Malformed Ordinal in the Location header.")
		T.Equal(event, nil)
	}()

	// Failure Condition 6: Malformed timestamp format.
	func() {
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
					resp.Header.Add("Location",
						"/v0/collection/key/events/type1/XXX/1")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		event, err := collection.innerAddEvent(keyFailure, "type1", nil, value)
		T.ExpectErrorMessage(err, "Malformed Timestamp in the Location header.")
		T.Equal(event, nil)
	}()

	// Failure Condition 7: Missing ETag header.
	func() {
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Etag")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		event, err := collection.innerAddEvent(keyFailure, "type1", nil, value)
		T.ExpectErrorMessage(err, "Missing ETag header.")
		T.Equal(event, nil)
	}()

	// Failure Condition 8: Malformed ETag header.
	func() {
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Etag")
					resp.Header.Add("Etag", "XXX")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		event, err := collection.innerAddEvent(keyFailure, "type1", nil, value)
		T.ExpectErrorMessage(err, "Malformed ETag header.")
		T.Equal(event, nil)
	}()

	// Success case 1: No timestamp
	func() {
		// Next we attempt to add an event to the item.
		event, err := collection.innerAddEvent(key, "type1", nil, value)
		T.ExpectSuccess(err)

		// All is good. Now attempt to get the event to ensure that it exists.
		var valueGet map[string]interface{}
		eventGet, err := collection.GetEvent(key, "type1", event.Timestamp,
			event.Ordinal, &valueGet)
		T.ExpectSuccess(err)
		T.Equal(valueGet, value)
		T.Equal(eventGet, event)
	}()

	// Success case 2: Given timestamp
	func() {
		// Next we attempt to add an event to the item.
		utime := time.Unix(10000, 100)
		event, err := collection.innerAddEvent(key, "type1", &utime, value)
		T.ExpectSuccess(err)

		// All is good. Now attempt to get the event to ensure that it exists.
		var valueGet map[string]interface{}
		eventGet, err := collection.GetEvent(key, "type1",
			event.Timestamp, event.Ordinal, &valueGet)
		T.ExpectSuccess(err)
		T.Equal(valueGet, value)
		T.Equal(eventGet, event)
	}()
}

func TestCollection_DeleteEvent(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Make sure that the item we are working with exists.
	key := "deleteevent"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Update(key, value)
	T.ExpectSuccess(err)

	// Next we attempt to add an event to the item.
	event, err := collection.AddEvent(key, "type1", value)
	T.ExpectSuccess(err)

	// Delete the event.
	err = collection.DeleteEvent(key, event.Type, event.Timestamp,
		event.Ordinal)
	T.ExpectSuccess(err)

	// Ensure that the event was deleted.
	var valueGet map[string]interface{}
	eventGet, err := collection.GetEvent(key, event.Type, event.Timestamp,
		event.Ordinal, &valueGet)
	T.ExpectErrorMessage(err, "404: Not found.")
	T.Equal(eventGet, nil)
}

func TestCollection_GetEvent(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// We use this value as the payload to the event.
	key := "getevent"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Update(key, value)
	T.ExpectSuccess(err)

	// Add the event.
	event, err := collection.AddEvent(key, "type1", value)
	T.ExpectSuccess(err)

	// Attempt to get the event and ensure that it gets the right data.
	var valueGet map[string]interface{}
	eventGet, err := collection.GetEvent(key, event.Type, event.Timestamp,
		event.Ordinal, &valueGet)
	T.ExpectSuccess(err)
	T.Equal(valueGet, value)
	T.Equal(eventGet, event)

	// Attempt the same get without a value.
	eventGet, err = collection.GetEvent(key, event.Type, event.Timestamp,
		event.Ordinal, nil)
	T.ExpectSuccess(err)
	T.Equal(eventGet, event)

	// Now get a key that won't exist.
	eventGet, err = collection.GetEvent("404_key", event.Type, event.Timestamp,
		event.Ordinal, nil)
	T.ExpectErrorMessage(err, "404: Not found.")
	T.Equal(eventGet, nil)
}

func TestCollection_UpdateEvent(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// We use this value as the payload to the event.
	key := "updateevent"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Update(key, value)
	T.ExpectSuccess(err)

	// Next we attempt to add an event to the item.
	event1, err := collection.AddEvent(key, "type1", value)
	T.ExpectSuccess(err)

	// And update the data.
	value2 := map[string]interface{}{
		"new_key1": "value1",
		"new_key2": "value2",
	}
	event2, err := collection.UpdateEvent(event1.Key, event1.Type,
		event1.Timestamp, event1.Ordinal, value2)
	T.ExpectSuccess(err)
	T.NotEqual(event2, event1)

	// Fetch the object and ensure that it is at event2.
	var valueGet map[string]interface{}
	eventGet, err := collection.GetEvent(key, "type1", event1.Timestamp,
		event1.Ordinal, &valueGet)
	T.ExpectSuccess(err)
	T.Equal(valueGet, value2)
	T.Equal(eventGet, event2)
}

func TestCollection_innerUpdateEvent(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// We use this value as the payload to the event.
	key := "innerupdateevent"
	keyFailure := "innerupdateevent_failure"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Update(key, value)
	T.ExpectSuccess(err)
	_, err = collection.Update(keyFailure, value)
	T.ExpectSuccess(err)

	// Next we attempt to add an event to the item.
	event, err := collection.AddEvent(key, "type1", value)
	T.ExpectSuccess(err)
	eventFailure, err := collection.AddEvent(key, "type1", value)
	T.ExpectSuccess(err)

	// Failure Condition 1: JSON Marshaling error.
	func() {
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&jsonMarshalError{}, nil)
		T.ExpectErrorMessage(err, "JSON MARSHAL ERROR")
		T.Equal(badEvent, nil)

	}()

	// Failure condition 2: client error.
	func() {
		var badValue map[string]interface{}
		defer func(s string) { client.APIHost = s }(client.APIHost)
		client.APIHost = "localhost:100000"
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&badValue, nil)
		T.ExpectErrorMessage(err, "dial tcp: invalid port 100000")
		T.Equal(badEvent, nil)
	}()

	// Failure Condition 3: Missing Location header.
	func() {
		badValue := map[string]string{"bad": "value"}
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&badValue, nil)
		T.ExpectErrorMessage(err, "Missing Location header.")
		T.Equal(badEvent, nil)
	}()

	// Failure Condition 4: Malformed location header.
	func() {
		badValue := map[string]string{"bad": "value"}
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
					resp.Header.Add("Location", "XXX")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&badValue, nil)
		T.ExpectErrorMessage(err, "Malformed Location header.")
		T.Equal(badEvent, nil)
	}()

	// Failure Condition 5: Malformed ordinal format.
	func() {
		badValue := map[string]string{"bad": "value"}
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
					resp.Header.Add("Location",
						"/v0/collection/key/events/type1/100/XXX")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&badValue, nil)
		T.ExpectErrorMessage(err, "Malformed Ordinal in the Location header.")
		T.Equal(badEvent, nil)
	}()

	// Failure Condition 6: Malformed timestamp format.
	func() {
		badValue := map[string]string{"bad": "value"}
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Location")
					resp.Header.Add("Location",
						"/v0/collection/key/events/type1/XXX/1")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&badValue, nil)
		T.ExpectErrorMessage(err, "Malformed Timestamp in the Location header.")
		T.Equal(badEvent, nil)
	}()

	// Failure Condition 7: Missing ETag header.
	func() {
		badValue := map[string]string{"bad": "value"}
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Etag")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&badValue, nil)
		T.ExpectErrorMessage(err, "Missing ETag header.")
		T.Equal(badEvent, nil)
	}()

	// Failure Condition 8: Malformed ETag header.
	func() {
		badValue := map[string]string{"bad": "value"}
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Etag")
					resp.Header.Add("Etag", "XXX")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		badEvent, err := collection.innerUpdateEvent(eventFailure.Key,
			eventFailure.Type, eventFailure.Timestamp, eventFailure.Ordinal,
			&badValue, nil)
		T.ExpectErrorMessage(err, "Malformed ETag header.")
		T.Equal(badEvent, nil)
	}()

	// Success: No timestamp
	func() {
		// And update the data.
		value2 := map[string]interface{}{
			"new_key1": "value1",
			"new_key2": "value2",
		}
		event2, err := collection.innerUpdateEvent(event.Key, event.Type,
			event.Timestamp, event.Ordinal, value2, nil)
		T.ExpectSuccess(err)
		T.NotEqual(event2, event)

		// Fetch the object and ensure that it is at event2.
		var valueGet map[string]interface{}
		eventGet, err := collection.GetEvent(event.Key, event.Type,
			event.Timestamp, event.Ordinal, &valueGet)
		T.ExpectSuccess(err)
		T.Equal(valueGet, value2)
		T.Equal(eventGet, event2)
	}()
}

func TestCollection_ListEvents(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Basic URL generation tests.
	func() {
		basePath := "gorc2-unittests/listevents/events/type"

		// Limit
		query := &ListEventsQuery{Limit: 100}
		iterator := collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?limit=100")

		// After (No ordinal)
		query = &ListEventsQuery{After: time.Unix(1000, 10000000)}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?afterEvent=1000010")

		// After (ordinal)
		query = &ListEventsQuery{
			After:        time.Unix(1000, 10000000),
			AfterOrdinal: 100000,
		}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?afterEvent=1000010%2F100000")

		// Before (No ordinal)
		query = &ListEventsQuery{Before: time.Unix(1000, 10000000)}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?beforeEvent=1000010")

		// Before (ordinal)
		query = &ListEventsQuery{
			Before:        time.Unix(1000, 10000000),
			BeforeOrdinal: 100000,
		}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?beforeEvent=1000010%2F100000")

		// End (No ordinal)
		query = &ListEventsQuery{End: time.Unix(1000, 10000000)}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?endEvent=1000010")

		// End (ordinal)
		query = &ListEventsQuery{
			End:        time.Unix(1000, 10000000),
			EndOrdinal: 100000,
		}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?endEvent=1000010%2F100000")

		// Start (No ordinal)
		query = &ListEventsQuery{Start: time.Unix(1000, 10000000)}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?startEvent=1000010")

		// Start (ordinal)
		query = &ListEventsQuery{
			Start:        time.Unix(1000, 10000000),
			StartOrdinal: 100000,
		}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?startEvent=1000010%2F100000")

		// Default
		iterator = collection.ListEvents("listevents", "type", nil)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath)
		query = &ListEventsQuery{}
		iterator = collection.ListEvents("listevents", "type", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, basePath+"?")
	}()

	// Purce, then re-create the item we are working with. This should remove
	// the entire event history on this object (and therefor remove previous
	// tests event additions)
	T.ExpectSuccess(collection.Purge("listevents"))
	value := map[string]string{"a": "b"}
	_, err := collection.Create("listevents", value)
	T.ExpectSuccess(err)

	// Load the data.
	allEvents := make([]*Event, 75)
	func() {
		// Load a ton of data into the system. We want 25 time stamps added,
		// with three ordinals each.
		var locker Locker
		for i := 0; i < len(allEvents); i += 3 {
			locker.Add(1)
			go func(i int) {
				defer locker.Done()
				var err error
				ts := time.Unix(int64(i)*10000, 0)
				for j := 0; j < 3; j++ {
					value := map[string]int{"index": i + j}
					allEvents[i+j], err = collection.AddEventWithTimestamp(
						"listevents", "type", ts, value)
					if err != nil {
						locker.Set(err)
						return
					}
				}
			}(i)
		}

		// Wait for the loaders to finish.
		locker.Wait()
		T.ExpectSuccess(locker.Err)
	}()

	// This test performs a query with the given options, expecting the given
	// Events to be returned without any errors. Since we know that the events
	// should be ordered in return this is simple.
	runTest := func(opts *ListEventsQuery, events []*Event) {
		iterator := collection.ListEvents("listevents", "type", opts)
		rindex := len(events) - 1
		for iterator.Next() {
			if rindex < 0 {
				T.Fatalf("Too many results returned: %d", len(iterator.results))
			}
			event, err := iterator.GetEvent(nil)
			T.ExpectSuccess(err)
			T.Equal(event, events[rindex])
			rindex--
		}
		T.ExpectSuccess(iterator.Error)
		T.Equal(len(iterator.results), len(events))
	}

	// Test 1: Default listing.
	query := &ListEventsQuery{Limit: 100}
	runTest(query, allEvents)

	// Test 2: After (no ordinal) (limit to 100 to reduce queries.)
	// 60 is the start of a time stamp range.
	query = &ListEventsQuery{
		Limit: 100,
		After: allEvents[60].Timestamp,
	}
	runTest(query, allEvents[60:])

	// Test 3: After (with ordinal) (limit to 100 to reduce queries.)
	// 61 is the middle of a range.
	query = &ListEventsQuery{
		Limit:        100,
		After:        allEvents[61].Timestamp,
		AfterOrdinal: allEvents[61].Ordinal,
	}
	runTest(query, allEvents[62:])

	// Test 4: Before (no ordinal) limit to 100 to reduce queries.)
	// 30 is the start of a range.
	query = &ListEventsQuery{
		Limit:  100,
		Before: allEvents[30].Timestamp,
	}
	runTest(query, allEvents[:30])

	// Test 5: Before (with ordinal) (limit to 100 to reduce queries.)
	// 31 is the middle of a range.
	query = &ListEventsQuery{
		Limit:         100,
		Before:        allEvents[31].Timestamp,
		BeforeOrdinal: allEvents[31].Ordinal,
	}
	runTest(query, allEvents[:31])

	// Test 6: End (no ordinal) limit to 100 to reduce queries.)
	// 30 is the start of a range so we should get back 30, 31, and 32.
	query = &ListEventsQuery{
		Limit: 100,
		End:   allEvents[30].Timestamp,
	}
	runTest(query, allEvents[:33])

	// Test 7: End (with ordinal) (limit to 100 to reduce queries.)
	// 31 is the middle of a range.
	query = &ListEventsQuery{
		Limit:      100,
		End:        allEvents[31].Timestamp,
		EndOrdinal: allEvents[31].Ordinal,
	}
	runTest(query, allEvents[:32])

	// Test 8: Start (no ordinal) limit to 100 to reduce queries.)
	// 60 is the start of a range so we should get back 60, 61, and 62.
	query = &ListEventsQuery{
		Limit: 100,
		Start: allEvents[60].Timestamp,
	}
	runTest(query, allEvents[60:])

	// Test 9: Start (with ordinal) (limit to 100 to reduce queries.)
	// 61 is the middle of a range.
	query = &ListEventsQuery{
		Limit:        100,
		Start:        allEvents[61].Timestamp,
		StartOrdinal: allEvents[61].Ordinal,
	}
	runTest(query, allEvents[61:])

	// Test 10: More complex range.
	query = &ListEventsQuery{
		Limit:        100,
		Start:        allEvents[31].Timestamp,
		StartOrdinal: allEvents[31].Ordinal,
		End:          allEvents[40].Timestamp,
		EndOrdinal:   allEvents[40].Ordinal,
	}
	runTest(query, allEvents[31:41])

	// Test 11: Another complex range.
	query = &ListEventsQuery{
		Limit:        100,
		After:        allEvents[31].Timestamp,
		AfterOrdinal: allEvents[31].Ordinal,
		Before:          allEvents[40].Timestamp,
		BeforeOrdinal:   allEvents[40].Ordinal,
	}
	runTest(query, allEvents[32:40])
}
