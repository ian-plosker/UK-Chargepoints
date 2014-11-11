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

func TestCollection_GetLinks(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Test that the URL is generated properly.
	func() {
		baseUrl := collection.Name + "/key1/relations/kind1"

		// Limit
		opts := &GetLinksQuery{Limit: 100}
		iterator := collection.GetLinks("key1", opts, "kind1")
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, baseUrl + "?limit=100")

		// Defaults
		iterator = collection.GetLinks("key1", nil, "kind1")
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, baseUrl)

		// Multiple kinds.
		iterator = collection.GetLinks("key1", nil, "kind1", "kind2", "kind3")
		T.Equal(iterator.client, client)
		T.Equal(iterator.next, baseUrl + "/kind2/kind3")
	}()

	// Add a main item that we can use with GetLinks().
	key := "getlinks"
	value := map[string]string{"test": "test"}
	T.ExpectSuccess(collection.Purge(key))
	_, err := collection.Create(key, value)
	T.ExpectSuccess(err)

	// Purge then Add 5 items with 5 sub items in parallel.
	locker := Locker{}
	for i := 0; i < 5; i++ {
		locker.Add(1)
		go func(i int) {
			defer locker.Done()
			key2 := fmt.Sprintf("getlinks_%d", i)
			locker.Set(collection.Purge(key2))
			_, err := collection.Create(key2, value)
			locker.Set(err)
			for j := 0; j < 5; j++ {
				key3 := fmt.Sprintf("%s_%d", key2, j)
				locker.Add(1)
				go func(key2, key3 string) {
					defer locker.Done()
					err := collection.Purge(key3)
					locker.Set(err)
					_, err = collection.Create(key3, value)
					locker.Set(err)
					err = collection.Link(key2, "kind2", collection.Name, key3)
					locker.Set(err)
				}(key2, key3)
			}
			err = collection.Link(key, "kind1", collection.Name, key2)
			locker.Set(err)
		}(i)
	}
	locker.Wait()
	T.ExpectSuccess(locker.Err)

	// Form a simple kind1 GetLinks call and ensure that 5 results are
	// returned.
	iterator := collection.GetLinks(key, nil, "kind1")
	seen := make(map[string]bool, 25)
	for iterator.Next() {
		item, err := iterator.Get(nil)
		T.ExpectSuccess(err)
		seen[item.Key] = true
	}
	T.ExpectSuccess(iterator.Error)
	T.Equal(len(seen), 5)
	for i := 0; i < 5; i++ {
		key2 := fmt.Sprintf("getlinks_%d", i)
		_, ok := seen[key2]
		T.Equal(ok, true)
	}

	// Do the same query but for a two layer setup.
	iterator = collection.GetLinks(key, nil, "kind1", "kind2")
	seen = make(map[string]bool, 50)
	for iterator.Next() {
		item, err := iterator.Get(nil)
		T.ExpectSuccess(err)
		seen[item.Key] = true
	}
	T.ExpectSuccess(iterator.Error)
	T.Equal(len(seen), 25)
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			key2 := fmt.Sprintf("getlinks_%d_%d", i, j)
			_, ok := seen[key2]
			T.Equal(ok, true)
		}
	}
}

func TestCollection_Link(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Delete then Add two items that we can link together.
	key1 := "link1"
	key2 := "link2"
	value := map[string]interface{}{"key": "value"}
	T.ExpectSuccess(collection.Purge(key1))
	item1, err := collection.Create(key1, value)
	T.ExpectSuccess(err)
	T.ExpectSuccess(collection.Purge(key2))
	item2, err := collection.Create(key2, value)
	T.ExpectSuccess(err)

	// Link item2 to item1 via "kind1"
	err = collection.Link(key1, "kind1", collection.Name, key2)
	T.ExpectSuccess(err)

	// Link item1 to item2 via "kind2"
	err = collection.Link(key2, "kind2", collection.Name, key1)
	T.ExpectSuccess(err)

	// Now we need to list the events to ensure that linking them together
	// worked.
	iterator := collection.GetLinks(key1, nil, "kind1")
	count := 0
	for iterator.Next() {
		if count != 0 {
			T.Fatalf("Too many results returned!")
		}
		i, err := iterator.Get(nil)
		T.ExpectSuccess(err)

		// Zero Updated (since the item returned from Create() won't have it.)
		i.Updated = time.Time{}
		T.Equal(i, item2)
		count += 1
	}
	T.ExpectSuccess(iterator.Error)
	T.Equal(count, 1)

	// Now we need to check that two layer graph traversal works as well.
	iterator = collection.GetLinks(key1, nil, "kind1", "kind2")
	count = 0
	for iterator.Next() {
		if count != 0 {
			T.Fatalf("Too many results returned!")
		}
		i, err := iterator.Get(nil)
		T.ExpectSuccess(err)

		// Zero Updated (since the item returned from Create() won't have it.)
		i.Updated = time.Time{}
		T.Equal(i, item1)
		count += 1
	}
	T.ExpectSuccess(iterator.Error)
	T.Equal(count, 1)
}

func TestCollection_Unlink(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Purge then Add two items that we can link together.
	key1 := "unlink1"
	key2 := "unlink2"
	value := map[string]interface{}{"key": "value"}
	T.ExpectSuccess(collection.Purge(key1))
	_, err := collection.Create(key1, value)
	T.ExpectSuccess(err)
	T.ExpectSuccess(collection.Purge(key2))
	item2, err := collection.Create(key2, value)
	T.ExpectSuccess(err)

	// Link item2 to item1 via "kind1"
	err = collection.Link(key1, "kind1", collection.Name, key2)
	T.ExpectSuccess(err)

	// Now we need to list the events to ensure that linking them together
	// worked.
	iterator := collection.GetLinks(key1, nil, "kind1")
	count := 0
	for iterator.Next() {
		if count != 0 {
			T.Fatalf("Too many results returned!")
		}
		i, err := iterator.Get(nil)
		T.ExpectSuccess(err)

		// Zero Updated (since the item returned from Create() won't have it.)
		i.Updated = time.Time{}
		T.Equal(i, item2)
		count += 1
	}
	T.ExpectSuccess(iterator.Error)
	T.Equal(count, 1)

	// Next we unlink the items.
	err = collection.Unlink(key1, "kind1", collection.Name, key2)
	T.ExpectSuccess(err)

	// Now wen ensure that there are no more links.
	iterator = collection.GetLinks(key1, nil, "kind1")
	for iterator.Next() {
		T.Fatalf("No results should be returned via this call.")
	}
	T.ExpectSuccess(iterator.Error)
}
