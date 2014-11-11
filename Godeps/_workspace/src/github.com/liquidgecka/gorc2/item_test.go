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
	"testing"

	"github.com/liquidgecka/testlib"
)

func TestEvent_Delete(t *testing.T) {
    T := testlib.NewT(t)
    defer T.Finish()
    client := cleanTestingClient(T)
    collection := client.Collection("gorc2-unittests")

    // Make sure that the item we are working with exists.
    key := "event_delete"
    value := map[string]interface{}{
        "key1": "value1",
        "key2": "value2",
    }
	T.ExpectSuccess(collection.Purge(key))
    _, err := collection.Update(key, value)
    T.ExpectSuccess(err)

    // Next we attempt to add an event to the item.
    event1, err := collection.AddEvent(key, "type2", value)
    T.ExpectSuccess(err)

    // Now we update the event.
    value2 := map[string]interface{}{
        "newkey1": "value1",
        "newkey2": "value2",
    }
    event2, err := event1.Update(value2)
    T.ExpectSuccess(err)

    // Attempt to delete the first event and ensure that we get the right error.
    err = event1.Delete()
    T.ExpectErrorMessage(err, "was not the most recent ref.")

    // All is good. Now attempt to get the event to ensure it still exists.
    var valueGet map[string]interface{}
    eventGet, err := collection.GetEvent(key, event2.Type,
        event2.Timestamp, event2.Ordinal, &valueGet)
    T.ExpectSuccess(err)
    T.Equal(valueGet, value2)
    T.Equal(eventGet, event2)

    // Now delete the Event with the most recent ref.
    T.ExpectSuccess(event2.Delete())

    // Ensure that the object is gone.
    eventGet, err = collection.GetEvent(key, event2.Type,
        event2.Timestamp, event2.Ordinal, &valueGet)
    T.ExpectErrorMessage(err, "404: Not found.")
    T.Equal(eventGet, nil)
}

func TestEvent_Update(t *testing.T) {
    T := testlib.NewT(t)
    defer T.Finish()
    client := cleanTestingClient(T)
    collection := client.Collection("gorc2-unittests")

    // We use this value as the payload to the event.
    key := "event_update"
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

    // Now attempt to update event 1.. this should fail.
    value3 := map[string]interface{}{
        "test": "test",
    }
    event3, err := event1.Update(value3)
    T.ExpectErrorMessage(err, "was not the most recent ref.")
    T.Equal(event3, nil)

    // Fetch the object and ensure that it is still at event2.
    var valueGet map[string]interface{}
    eventGet, err := collection.GetEvent(key, "type1", event1.Timestamp,
        event1.Ordinal, &valueGet)
    T.ExpectSuccess(err)
    T.Equal(valueGet, value2)
    T.Equal(eventGet, event2)

    // Now try updating event2 which should work.
    event3, err = event2.Update(value3)
    T.ExpectSuccess(err)

    // Fetch the object and ensure that it now event3.
    valueGet = map[string]interface{}{}
    eventGet, err = collection.GetEvent(key, "type1", event1.Timestamp,
        event1.Ordinal, &valueGet)
    T.ExpectSuccess(err)
    T.Equal(valueGet, value3)
    T.Equal(eventGet, event3)
}

func TestItem_Delete(t *testing.T) {
    T := testlib.NewT(t)
    defer T.Finish()
    client := cleanTestingClient(T)
    collection := client.Collection("gorc2-unittests")

    // Add an item for us to delete.
    key := "conditional_delete"
    value := map[string]interface{}{
        "key1": "value1",
        "key2": "value2",
    }
    item1, err := collection.Create(key, value)
    T.ExpectSuccess(err)

    // Update the object to a new ref.
    value2 := map[string]interface{}{
        "new_key1": "value1",
        "new_key2": "value2",
    }
    item2, err := collection.Update(key, value2)

    // Try to delete the first object and ensure that it fails.
    err = item1.Delete()
    T.ExpectErrorMessage(err, "was not the most recent ref.")

    // Get the object and ensure that it is the same as item2.
    itemGet, err := collection.Get(key, nil)
    T.ExpectSuccess(err)
    T.Equal(itemGet, item2)

    // Conditionally delete item2 and ensure that it works.
    err = item2.Delete()
    T.ExpectSuccess(err)

    // Ensure that the object was actually deleted.
    itemGet, err = collection.Get(key, nil)
    T.ExpectErrorMessage(err, "404: Not found.")
    T.Equal(itemGet, nil)
}

func TestItem_Update(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()
	client := cleanTestingClient(T)
	collection := client.Collection("gorc2-unittests")

	// Purge the keys we are going to be using.
	collection.Purge("conditionalupdate_test")

	// Create the item.
	value1 := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	item1, err := collection.Create("conditionalupdate_test", value1)
	T.ExpectSuccess(err)

	// Verify that the item was added.
	var valueGet map[string]string
	itemGet, err := collection.Get("conditionalupdate_test", &valueGet)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item1)
	T.Equal(valueGet, value1)

	// Now attempt to update the item.
	value2 := map[string]string{
		"new_key1": "new_value1",
		"new_key2": "new_value2",
	}
	item2, err := collection.Update("conditionalupdate_test", value2)
	T.ExpectSuccess(err)
	T.ExpectSuccess(err)

	// Verify that the item was updated.
	valueGet = map[string]string{}
	itemGet, err = collection.Get("conditionalupdate_test", &valueGet)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item2)
	T.Equal(valueGet, value2)

	// Now test that a conditional update of the first item fails.
	value3 := map[string]string{
		"bad_key1": "value1",
		"bad_key2": "value2",
	}
	item3, err := item1.Update(value3)
	T.ExpectError(err)
	T.Equal(nil, item3)

	// Verify that the data remains the same as item2.
	valueGet = map[string]string{}
	itemGet, err = collection.Get("conditionalupdate_test", &valueGet)
	T.ExpectSuccess(err)
	T.Equal(itemGet, item2)
	T.Equal(valueGet, value2)
}
