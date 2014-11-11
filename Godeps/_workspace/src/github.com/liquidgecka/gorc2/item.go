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
	"encoding/json"
	"time"
)

//
// Collection
//

// Represents a Collection in Orchestrate.
type Collection struct {
	// The unique name of this collection.
	Name string

	// A reference back to the Client that created this Collection.
	client *Client
}

//
// Event
//

// Represents a single Event in a Collection.
type Event struct {
	// The collection that this Event is attached too.
	Collection *Collection

	// The Item Key that this Event is attached too.
	Key string

	// The update Ordinal for this event.
	Ordinal int64

	// The Reference number for this specific event.
	Ref string

	// The user supplied Timestamp associated with this event.
	Timestamp time.Time

	// The user supplied Type associated with this event.
	Type string

	// The raw JSON value.
	Value json.RawMessage
}

// FIXME

// Deletes the Event if it is the most recent event for the given key, time
// stamp, and ordinal pairing. This will return an error if the event has
// been updated via a prior call to Update() or Delete().
func (e *Event) Delete() error {
	headers := map[string]string{"If-Match": `"` + e.Ref + `"`}
	path := fmt.Sprintf("%s/%s/events/%s/%d/%d?purge=true",
		e.Collection.Name, e.Key, e.Type, e.Timestamp.UnixNano()/1000000,
		e.Ordinal)
	_, err := e.Collection.client.emptyReply("DELETE", path, headers, nil, 204)
	if err != nil {
		if _, ok := err.(PreconditionFailedError); ok {
			err = NotMostRecentError(e.Ref)
		}
	}
	return err
}

// Unmarshal's the data from 'Value' into the given item.
func (e *Event) Unmarshal(value interface{}) error {
	return json.Unmarshal(e.Value, value)
}

// Updates this event if it represents the most recent event for the key,
// timestamp, and ordinal pairing. This will return an error if the event has
// already been updated via a prior call to Event.Update().
func (e *Event) Update(value interface{}) (*Event, error) {
	headers := map[string]string{
		"If-Match":     `"` + e.Ref + `"`,
		"Content-Type": "application/json",
	}
	event, err := e.Collection.innerUpdateEvent(e.Key, e.Type, e.Timestamp,
		e.Ordinal, value, headers)
	if err != nil {
		if _, ok := err.(PreconditionFailedError); ok {
			err = NotMostRecentError(e.Ref)
		}
	}
	return event, err
}

//
// Item
//

// Stores information about a single Item from the Key Value part of a
// Collection.
type Item struct {
	// The Collection that houses this item.
	Collection *Collection

	// Distance is set on queries that include geospacial search. If
	// this field is non zero then Score will be zero.
	// See http://orchestrate.io/blog/2014/10/08/geospatial-search/
	Distance float32

	// The Key used to store this item within its collection.
	Key string

	// The Ref value for this item which uniquely identifies its version.
	Ref string

	// For Search results this will be populated with the score returned from
	// Orchestrate. Higher numbers mean better matches.
	Score float32

	// Set to true if this item represents a "Tombstone", or delete operation.
	// If this is set then other fields, like Value might not be set at all.
	// Only calls to History calls will set this field.
	Tombstone bool

	// The time that this item was created or updated in Orchestrate. This is
	// only populated on History calls at the moment.
	Updated time.Time

	// The raw JSON value returned by Orchestrate. To decode this value into
	// a structure use the Unmarshal() call.
	Value json.RawMessage
}

// Delete the Item from the collection if it represents the most recent
// 'Ref' associated with the key. If the key has been updated at some point
// after this item then this call will fail, reduring a NotMostRecentError
// object.
func (i *Item) Delete() error {
	headers := map[string]string{"If-Match": `"` + i.Ref + `"`}
	path := i.Collection.Name + "/" + i.Key
	_, err := i.Collection.client.emptyReply("DELETE", path, headers, nil, 204)
	if err != nil {
		if _, ok := err.(PreconditionFailedError); ok {
			err = NotMostRecentError(i.Ref)
		}
	}
	return err
}

// This will take the raw JSON data returned from Orchestrate and Unmarshal it
// into the given object.
func (i *Item) Unmarshal(value interface{}) error {
	return json.Unmarshal(i.Value, value)
}

// Updates this Item in the key value store if it is the most recent 'Ref'
// associated with the given key. If the given Item's Ref field does not match
// the most recently updated item then this call will return a
// NotMostRecentError type, and no change will be made in the data store.
func (i *Item) Update(value interface{}) (*Item, error) {
	headers := map[string]string{"If-Match": `"` + i.Ref + `"`}
	item, err := i.Collection.innerPut(i.Key, headers, value)
	if err != nil {
		if _, ok := err.(PreconditionFailedError); ok {
			err = NotMostRecentError(i.Key)
		}
	}
	return item, err
}
