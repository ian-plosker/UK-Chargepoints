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

package gorc2_test

import (
	"github.com/orchestrate-io/gorc2"
)

// Documents the basic CRUD operations of the Key-Value store in Orchestrate.
func Example_keyValue() {
	// Replace the key and collection name with a valid value obtained from
	// https://dashboard.orchestrate.io
	client := gorc2.NewClient("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	collection := client.Collect("collectionname")

	// Create an item in the collection with the key "key1". This operation
	// will return an error if the key already exists.
	valueCreate := map[string]int {"version": 1}
	itemCreate, err := collection.Create("key1", valueCreate)
	if err != nil {
		panic(err)
	}

	// Retrieving the object is fairly simple. In this case, the returned
	// Item should be the same as ItemCreate.
	valueGet := make(map[string]int, 10)
	_, err := collection.Get("key1", &valueGet)
	if err != nil {
		panic(err)
	}

	// Update the item with a new value. Update will create a value if it did
	// not already exist.
	valueUpdate := map[string]int {"version": 2}
	_, err := collection.Update("key1", valueUpdate)
	if err != nil {
		panic(err)
	}

	// And finally, deleting the object. Note that the object does not need
	// to exist for the delete to succeed.
	if err := collection.Delete("key1"); err != nil {
		panic(err)
	}
}

// Documents the ability for Orchestrate to Conditionally update or delete
// objects. Conditions allow the developer to ensure that there is not an
// update race condition when changing data.
func Example_conditionalOperations() {
	// Replace the key and collection name with a valid value obtained from
	// https://dashboard.orchestrate.io
	client := gorc2.NewClient("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	collection := client.Collect("collectionname")

	// We start by making a item, then updating it so we have two "generations"
	// of the item to work with.
	valueOld := map[string]int {"version": 1}
	itemOld, err := collection.Create("key1", valueOld)
	if err != nil {
		panic(err)
	}
	valueCurrent := map[string]int {"version": 2}
	itemCurrent, err := collection.Update("key1", valueCurrent)
	if err != nil {
		panic(err)
	}

	// Now we can demonstrate that an update on the older item will fail, and
	// we can know that via the type on the error returned. If it is anything
	// other than a gorc2.NotMostRecentError then something else failed along
	// the way.
	valueNext := map[string]int {"version": 3}
	_, err := itemOld.ConditionalUpdate(valueNext)
	if _, ok := err.(gorc2.NotMostRecentError); !ok {
		panic(err)
	}

	// Likewise we can use the ConditionalDelete function in the same way. This
	// will only delete the Item if it is the most recent item into the store.
	err = itemOld.ConditionalDelete()
	if _, ok := err.(gorc2.NotMostRecentError); !ok {
		panic(err)
	}
}

// Orchestrate provides history on items that have been updated. Each
// object is given a 'ref' which allows specific item parsing. This example
// shows how to work with history of an item.
func Example_history() {
	// Replace the key and collection name with a valid value obtained from
	// https://dashboard.orchestrate.io
	client := gorc2.NewClient("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	collection := client.Collect("collectionname")

	// For this example we are going to assume that an item with the "key1"
	// Key has gone through the following history:
	//  1. Created with {"iteration": 1}
	//  2. Deleted
	//  3. Created with {"iteration": 2}
	//  4. Updated with {"iteration": 3}

	// Start by getting the history listing. This is done via an iterator
	// which will walk through the results and automatically fetch the
	// next items.
	iterator := collection.History("key1")
	for iterator.Next() {
		// The item returned from Get() will return the following on each
		// pass through the loop:
		//
		// First pass: item.Tombstone == false, value = {"iteration": 3}
		// Second pass: item.Tombstone == false, value = {"iteration": 2}
		// Third pass: item.Tombstone == true, value is untoucned.
		// Fourth pass: item.Tombstone == false, value = {"iteration": 1}
		//
		// After this Next() will return false and the loop will end.
		var value map[string]int
		item, err := iterator.Get(&value)
		if err != nil {
			panic(err)
		}
	}

	// After iteration we need to check if it stopped iterating due to an
	// error.
	if iterator.Error != nil {
		panic(iterator.Error)
	}

	// If you know the specific 'ref' for an item in this key you can
	// fetch it directly.
	specItem, err := collection.GetRef("key1", "specificRef", nil)
	if err != nil {
		panic(err)
	}
}
