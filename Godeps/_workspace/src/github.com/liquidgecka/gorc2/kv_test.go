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
	"net/http"
	"testing"
	"time"

	"github.com/liquidgecka/testlib"
)

func TestCollection_Create(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Purge "create_success" if it exists.
	collection.Purge("create_success")

	// We use this value as the payload to the request.
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	// Success
	item, err := collection.Create("create_success", value)
	T.ExpectSuccess(err)
	T.Equal(item.Key, "create_success")
	T.Equal(item.Collection, collection)

	// Make sure that a second fall fails.
	item, err = collection.Create("create_success", value)
	T.ExpectError(err)
	T.Equal(err, AlreadyExistsError("create_success"))
	T.Equal(item, nil)
}

func TestCollection_Delete(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Add an item for us to delete.
	key := "delete_test"
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Create(key, value)
	T.ExpectSuccess(err)

	// Get the object to ensure that it exists.
	_, err = collection.Get(key, nil)
	T.ExpectSuccess(err)

	// Delete the object and expect it to work.
	T.ExpectSuccess(collection.Delete(key))

	// Get the object and ensure that it returns an error.
	_, err = collection.Get(key, nil)
	T.ExpectError(err)
}

func TestCollection_Purge(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Add an item for us to delete.
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	_, err := collection.Create("purge_test", value)
	T.ExpectSuccess(err)

	// Get the object to ensure that it exists.
	_, err = collection.Get("purge_test", nil)
	T.ExpectSuccess(err)

	// Delete the object and expect it to work.
	T.ExpectSuccess(collection.Purge("purge_test"))

	// Get the object and ensure that it returns an error.
	_, err = collection.Get("purge_test", nil)
	T.ExpectError(err)

	// TODO: Make sure that the item is removed from history as well.
}

func TestCollection_Get(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Purge the two keys we use.
	collection.Purge("get_test")
	collection.Purge("get_404")

	// Add an item for us to get.
	value := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	item, err := collection.Create("get_test", value)
	T.ExpectSuccess(err)

	// Get the object and encode the value into something we can compare.
	var valueGet map[string]string
	itemGet, err := collection.Get("get_test", &valueGet)
	T.ExpectSuccess(err)

	// Ensure that the returned item is exactly the same, and that the value
	// was returned exactly as inserted.
	T.Equal(itemGet, item)
	T.Equal(valueGet, value)

	// Lastly we ensure that getting an item that doesn't exist returns
	// the right kind of error.
	item, err = collection.Get("get_404", nil)
	T.ExpectError(err)
	T.Equal(item, nil)
	T.Equal(err, NotFoundError("404: Not found."))
}

func TestCollection_GetRef(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Purge the keys we are going to be using.
	collection.Purge("getref_test")
	collection.Purge("getref_404")

	// Store the values and items from each iteration.
	values := make([]map[string]int, 10)
	items := make([]*Item, 10)

	// Start off by adding 10 iterations to the same key.
	for i := 0; i < 10; i++ {
		var err error
		values[i] = map[string]int{
			"iteration": i,
		}
		items[i], err = collection.Update("getref_test", values[i])
		T.ExpectSuccess(err)
	}

	// Now, walk through the keys in order attempting to get the specific
	// ref for each item.
	for i := 0; i < 10; i++ {
		var value map[string]int
		item, err := collection.GetRef("getref_test", items[i].Ref, &value)
		T.ExpectSuccess(err)
		T.Equal(item, items[i])
		T.Equal(value, values[i])
	}

	// And lastly we ensure that an empty ref returns the most recent update.
	var value map[string]int
	item, err := collection.GetRef("getref_test", "", &value)
	T.ExpectSuccess(err)
	T.Equal(item, items[len(items)-1])
	T.Equal(value, values[len(items)-1])

	// Ensure that an error is thrown if the object doesn't exist.
	item, err = collection.Get("getref_404", nil)
	T.ExpectError(err)
	T.Equal(item, nil)
	T.Equal(err, NotFoundError("404: Not found."))

	// And lastly ensure that an error is thrown if the Content-Location
	// header is missing.
	func() {
		defer func(c *http.Client) { client.HTTPClient = c }(client.HTTPClient)
		client.HTTPClient = &http.Client{
			Transport: &testRoundTripper{
				f: func(resp *http.Response) {
					resp.Header.Del("Content-Location")
				},
				RoundTripper: http.DefaultTransport,
			},
		}
		item, err := collection.GetRef("getref_test", "", &value)
		T.ExpectError(err)
		T.ExpectErrorMessage(err, "Missing Content-Location header.")
		T.Equal(item, nil)
	}()
}

func TestCollection_History(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Test the basic query generation logic.
	func() {
		baseUrl := collection.Name + "/history/refs"

		// Limit
		opts := &HistoryQuery{Limit: 100}
		iterator := collection.History("history", opts)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, baseUrl+"?limit=100")

		// Offset
		opts = &HistoryQuery{Offset: 100}
		iterator = collection.History("history", opts)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, baseUrl+"?offset=100")

		// BeforeKey
		opts = &HistoryQuery{Values: true}
		iterator = collection.History("history", opts)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, baseUrl+"?values=true")

		// Defaults
		iterator = collection.History("history", nil)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, baseUrl+"?")
	}()

	// Purge the history on the key, then loop through adding 25 iterations
	// of the key so we have known history to compare against.
	T.ExpectSuccess(collection.Purge("history"))
	items := make([]*Item, 25)
	values := make([]map[string]int, len(items))
	var err error
	for i := len(items) - 1; i >= 0; i -= 1 {
		// Ensure that every 7 items we return a tombstone.
		if i%7 == 3 {
			err = collection.Delete("history")
			T.ExpectSuccess(err)
			items[i] = nil
			values[i] = nil
		} else {
			values[i] = map[string]int{"version": i}
			items[i], err = collection.Update("history", values[i])
			T.ExpectSuccess(err)
		}
	}

	// Query the history using the given query object and ensure that the
	// results match the passed in slice.
	test := func(q *HistoryQuery, items []*Item, values []map[string]int) {
		index := 0
		iterator := collection.History("history", q)
		for iterator.Next() {
			if index > len(items) {
				t.Fatalf("Too many results returned.")
			}
			item, err := iterator.Get(nil)
			T.ExpectSuccess(err)
			if items[index] == nil {
				T.Equal(item.Tombstone, true)
				T.Equal(item.Value, nil)
			} else {
				// Ignore the "RefTime" field since it is not returned via
				// the Get call earlier.
				item.Updated = time.Time{}
				T.Equal(item, items[index])
				value := map[string]int{}
				T.ExpectSuccess(item.Unmarshal(&value))
				T.Equal(value, values[index])
			}
			index += 1
		}
		T.ExpectSuccess(iterator.Error)
		T.Equal(index, len(items))
	}

	// Run the test on the full dataset.
	test(&HistoryQuery{Limit: 100, Values: true}, items, values)

	// Run the test with an offset of 20.
	test(&HistoryQuery{Offset: 20, Values: true}, items[20:], values[20:])
}

func TestCollection_List(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Limit
	query := &ListQuery{Limit: 100}
	iterator := collection.List(query)
	T.Equal(iterator.client, client)
	T.Equal(iterator.next, "gorc2-unittests?limit=100")

	// AfterKey
	query = &ListQuery{AfterKey: "TEST"}
	iterator = collection.List(query)
	T.Equal(iterator.client, client)
	T.Equal(iterator.next, "gorc2-unittests?afterKey=TEST")

	// BeforeKey
	query = &ListQuery{BeforeKey: "TEST"}
	iterator = collection.List(query)
	T.Equal(iterator.client, client)
	T.Equal(iterator.next, "gorc2-unittests?beforeKey=TEST")

	// EndKey
	query = &ListQuery{EndKey: "TEST"}
	iterator = collection.List(query)
	T.Equal(iterator.client, client)
	T.Equal(iterator.next, "gorc2-unittests?endKey=TEST")

	// StartKey
	query = &ListQuery{StartKey: "TEST"}
	iterator = collection.List(query)
	T.Equal(iterator.client, client)
	T.Equal(iterator.next, "gorc2-unittests?startKey=TEST")

	// Defaults
	iterator = collection.List(nil)
	T.Equal(iterator.client, client)
	T.Equal(iterator.next, "gorc2-unittests")

	// FIXME: Ensure that listing actually performs the expected work.
}

func TestCollection_Search(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests-search")
	PurgeTheTestCollection(T, collection)

	// Basic URL generation tests.
	func() {
		// Limit
		query := &SearchQuery{Limit: 100}
		iterator := collection.Search("test", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, "gorc2-unittests-search?limit=100&query=test")

		// Offset
		query = &SearchQuery{Offset: 10000}
		iterator = collection.Search("test", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, "gorc2-unittests-search?offset=10000&query=test")

		// Sort
		query = &SearchQuery{Sort: "SORT"}
		iterator = collection.Search("test", query)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, "gorc2-unittests-search?query=test&sort=SORT")

		// Default
		iterator = collection.Search("test", nil)
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, "gorc2-unittests-search?query=test")
	}()

	// Load a bunch of data in so we can search in the following tests.
	func() {
		var locker Locker
		loadChan := make(chan map[string]interface{}, 100)
		go func() {
			// The first set of items will have fields containing "str1"
			for i := 0; i < 50; i++ {
				value := map[string]interface{}{
					fmt.Sprintf("item%d", i): map[string]interface{}{
						"field": "contains str1 string",
					},
				}
				loadChan <- value
			}

			// Next we create items that do NOT have the str1 field.
			for i := 0; i < 50; i++ {
				value := map[string]interface{}{
					fmt.Sprintf("item%d", i+50): map[string]interface{}{
						"field": "does not match",
					},
				}
				loadChan <- value
			}

			// Finished!
			close(loadChan)
		}()

		// Start 25 loader goroutines.
		for i := 0; i < 25; i++ {
			locker.Add(1)
			go func() {
				defer locker.Done()
				for data := range loadChan {
					for key, value := range data {
						if locker.IsSet() {
							continue
						}
						_, err := collection.Update(key, value)
						locker.Set(err)
					}
				}
			}()
		}

		// Wait on to goroutines
		locker.Wait()
		T.ExpectSuccess(locker.Err)

	}()

	// Perform a search on items containing "str1". Since indexing might take
	// a period of time we have to check the results until all of them appear
	// or a given timeout happens.
	var seen map[string]map[string]string
	T.TryUntil(func() bool {
		seen = make(map[string]map[string]string, 50)
		iterator := collection.Search("str1", &SearchQuery{Limit: 20})
		for iterator.Next() {
			value := map[string]string{}
			item, err := iterator.Get(&value)
			T.ExpectSuccess(err)
			T.NotEqual(item, nil)
			T.NotEqual(item.Score, 0.0)
			seen[item.Key] = value
		}
		T.ExpectSuccess(iterator.Error)
		return len(seen) == 50
	}, time.Second*5)

	// Check that the expected items were returned.
	for i := 0; i < 50; i++ {
		want := map[string]string{"field": "contains str1 string"}
		key := fmt.Sprintf("item%d", i)
		have, ok := seen[key]
		T.Equal(ok, true)
		T.Equal(have, want)
	}
}

func TestCollection_Update(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Purge the keys we are going to be using.
	collection.Purge("update_test")

	value := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// Test that Update creates the item if it doesn't exist.
	item, err := collection.Update("update_test", value)
	T.ExpectSuccess(err)

	// Verify that the item was added.
	var valueGet map[string]string
	itemGet, err := collection.Get("update_test", &valueGet)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item)
	T.Equal(valueGet, value)

	// Now attempt to update the item.
	value = map[string]string{
		"new_key1": "new_value1",
		"new_key2": "new_value2",
	}
	item, err = collection.Update("update_test", value)
	T.ExpectSuccess(err)
	T.ExpectSuccess(err)

	// Verify that the item was updated.
	valueGet = map[string]string{}
	itemGet, err = collection.Get("update_test", &valueGet)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item)
	T.Equal(valueGet, value)
}

func TestCollection_innerPut(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Purge the keys we are going to be using.
	collection.Purge("innerPut_success")
	collection.Purge("innerPut_failure1")
	collection.Purge("innerPut_failure2")
	collection.Purge("innerPut_failure3")

	// We use this value as the payload to the request.
	value := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	// Success
	func() {
		_, err := collection.innerPut("innerPut_success", nil, value)
		T.ExpectSuccess(err)
	}()

	// Failure case 1: JSON encoding fails.
	func() {
		badValue := &jsonMarshalError{}
		item, err := collection.Create("innerPut_failure1", badValue)
		T.ExpectError(err)
		T.ExpectErrorMessage(err, "JSON MARSHAL ERROR")
		T.Equal(item, nil)
	}()

	// Failure case 2: Non 204 HTTP response.
	func() {
		defer func(s string) { client.APIHost = s }(client.APIHost)
		client.APIHost = "localhost:100000"
		item, err := collection.Create("innerPut_failure2", value)
		T.ExpectError(err)
		T.ExpectErrorMessage(err, "dial tcp: invalid port 100000")
		T.Equal(item, nil)
	}()

	// Failure case 3: Missing Location header.
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
		item, err := collection.Create("innerPut_failure3", value)
		T.ExpectError(err)
		T.ExpectErrorMessage(err, "Missing Location header.")
		T.Equal(item, nil)
	}()
}
