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
	"fmt"
	"testing"
	"time"

	"github.com/liquidgecka/testlib"
)

func TestIterator_Next(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-iterator-unittests")
	PurgeTheTestCollection(T, collection)

	// False condition 1: No results.
	func() {
		iterator := collection.List(nil)
		T.Equal(iterator.Next(), false)
		T.ExpectSuccess(iterator.Error)
	}()

	// False condition 2: Iterator is already done.
	func() {
		iterator := &Iterator{done: true}
		T.Equal(iterator.Next(), false)
	}()

	// False condition 2: An error was returned.
	func() {
		iterator := &Iterator{Error: fmt.Errorf("Expected")}
		T.Equal(iterator.Next(), false)
	}()

	// False condition 3: Next link is empty.
	func() {
		iterator := &Iterator{next: ""}
		T.Equal(iterator.Next(), false)
	}()

	// False condition 4: Error fetching data.
	func() {
		defer func(s string) { client.APIHost = s }(client.APIHost)
		client.APIHost = "localhost:100000"
		iterator := collection.List(nil)
		T.Equal(iterator.Next(), false)
		T.ExpectErrorMessage(iterator.Error,
			"dial tcp: invalid port 100000")
	}()

	// For all the following tests we want the data to be loaded. As such we
	// add 10 items to the collection.
	for i := 0; i < 10; i++ {
		value := map[string]int{"iteration": i}
		key := fmt.Sprintf("iteration%d", i)
		_, err := collection.Create(key, value)
		T.ExpectSuccess(err)
	}

	// True condition 1: Several iteration calls.
	func() {
		// The ordering is not ensured so we make a map to check off the ones
		// returned.
		seen := make(map[string]map[string]int, 10)
		iterator := collection.List(&ListQuery{Limit: 5})
		for iterator.Next() {
			value := map[string]int{}
			item, err := iterator.Get(&value)
			T.ExpectSuccess(err)
			T.NotEqual(item, nil)
			seen[item.Key] = value
		}
		T.ExpectSuccess(iterator.Error)

		// Check that all the items were seen.
		T.Equal(len(seen), 10)
		for i := 0; i < 10; i++ {
			want := map[string]int{"iteration": i}
			key := fmt.Sprintf("iteration%d", i)
			have, ok := seen[key]
			T.Equal(ok, true)
			T.Equal(have, want)
		}
	}()
}

func TestIterator_NextWithError(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// done
	iterator := collection.List(nil)
	iterator.done = true
	done, err := iterator.NextWithError()
	T.Equal(done, false)
	T.ExpectSuccess(err)

	// error
	iterator.done = false
	iterator.Error = fmt.Errorf("EXPECTED")
	done, err = iterator.NextWithError()
	T.Equal(done, false)
	T.ExpectError(iterator.Error)
}

func TestIterator_Get(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Define two basic items.
	item1 := &Item{
		Collection: collection,
		Key:        "item1_key",
		Ref:        "0123456789abcdef",
		Score:      1.1,
		Tombstone:  false,
		Updated:    time.Unix(100, 100*1000000),
		Value:      []byte(`{"a": "b"}`),
	}
	item2 := &Item{
		Collection: collection,
		Key:        "item2_key",
		Ref:        "fedcba9876543210",
		Score:      -2.0,
		Tombstone:  true,
		Updated:    time.Unix(500, 500*1000000),
		Value:      []byte(`{"b": "a"}`),
	}

	// Setup some basic data.
	iterator := &Iterator{
		client:         client,
		iteratingItems: true,
	}
	iterator.results = []*jsonListItem{
		&jsonListItem{
			Path: jsonPath{
				Collection: "gorc2-unittests",
				Key:        item1.Key,
				Ref:        item1.Ref,
				Tombstone:  item1.Tombstone,
			},
			RefTime: item1.Updated.UnixNano() / 1000000,
			Score:   item1.Score,
			Value:   item1.Value,
		},
		&jsonListItem{
			Path: jsonPath{
				Collection: "gorc2-unittests",
				Key:        item2.Key,
				Ref:        item2.Ref,
				Tombstone:  item2.Tombstone,
			},
			RefTime: item2.Updated.UnixNano() / 1000000,
			Score:   item2.Score,
			Value:   item2.Value,
		},
	}
	iterator.index = 0

	// No json decoding.
	itemGet, err := iterator.Get(nil)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item1)

	// json decoding.
	var retValue map[string]string
	expValue := map[string]string{"a": "b"}
	itemGet, err = iterator.Get(&retValue)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item1)
	T.Equal(retValue, expValue)

	// Next index.
	iterator.index = 1
	itemGet, err = iterator.Get(nil)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item2)

	// json decoding.
	retValue = map[string]string{}
	expValue = map[string]string{"b": "a"}
	itemGet, err = iterator.Get(&retValue)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item2)
	T.Equal(retValue, expValue)

	// Can not decode.
	var decErr jsonUnmarshalError
	itemGet, err = iterator.Get(&decErr)
	T.ExpectErrorMessage(err, "JSON UNMARSHAL ERROR")
	T.Equal(itemGet, item2)

	// Is not an Item iterator.
	iterator.iteratingItems = false
	itemGet, err = iterator.Get(nil)
	T.ExpectErrorMessage(err, "Not an Item Iterator.")
	T.Equal(itemGet, nil)
}

func TestIterator_GetEvent(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Define two basic events.
	event1 := &Event{
		Collection: collection,
		Key:        "event1_key",
		Ordinal:    100,
		Ref:        "0123456789abcdef",
		Timestamp:  time.Unix(100, 100*1000000),
		Type:       "type1",
		Value:      []byte(`{"a": "b"}`),
	}
	event2 := &Event{
		Collection: collection,
		Key:        "event2_key",
		Ordinal:    5,
		Ref:        "fedcba9876543210",
		Timestamp:  time.Unix(500, 500*1000000),
		Type:       "type2",
		Value:      []byte(`{"b": "a"}`),
	}

	// Setup some basic data.
	iterator := &Iterator{
		client:          client,
		iteratingEvents: true,
	}
	iterator.results = []*jsonListItem{
		&jsonListItem{
			Path: jsonPath{
				Collection: "gorc2-unittests",
				Key:        event1.Key,
				Ordinal:    event1.Ordinal,
				Ref:        event1.Ref,
				Type:       event1.Type,
				Timestamp:  event1.Timestamp.UnixNano() / 1000000,
			},
			Ordinal:   event1.Ordinal,
			Timestamp: event1.Timestamp.UnixNano() / 1000000,
			Value:     event1.Value,
		},
		&jsonListItem{
			Path: jsonPath{
				Collection: "gorc2-unittests",
				Key:        event2.Key,
				Ordinal:    event2.Ordinal,
				Ref:        event2.Ref,
				Type:       event2.Type,
				Timestamp:  event2.Timestamp.UnixNano() / 1000000,
			},
			Ordinal:   event2.Ordinal,
			Timestamp: event2.Timestamp.UnixNano() / 1000000,
			Value:     event2.Value,
		},
	}
	iterator.index = 0

	// No json decoding.
	eventGet, err := iterator.GetEvent(nil)
	T.ExpectSuccess(err)
	T.Equal(eventGet, event1)

	// json decoding.
	var retValue map[string]string
	expValue := map[string]string{"a": "b"}
	eventGet, err = iterator.GetEvent(&retValue)
	T.ExpectSuccess(err)
	T.Equal(eventGet, event1)
	T.Equal(retValue, expValue)

	// Next index.
	iterator.index = 1
	eventGet, err = iterator.GetEvent(nil)
	T.ExpectSuccess(err)
	T.Equal(eventGet, event2)

	// json decoding.
	retValue = map[string]string{}
	expValue = map[string]string{"b": "a"}
	eventGet, err = iterator.GetEvent(&retValue)
	T.ExpectSuccess(err)
	T.Equal(eventGet, event2)
	T.Equal(retValue, expValue)

	// Can not decode.
	var decErr jsonUnmarshalError
	eventGet, err = iterator.GetEvent(&decErr)
	T.ExpectErrorMessage(err, "JSON UNMARSHAL ERROR")
	T.Equal(eventGet, event2)

	// Is not an Item iterator.
	iterator.iteratingEvents = false
	eventGet, err = iterator.GetEvent(nil)
	T.ExpectErrorMessage(err, "Not an Event Iterator.")
	T.Equal(eventGet, nil)
}
