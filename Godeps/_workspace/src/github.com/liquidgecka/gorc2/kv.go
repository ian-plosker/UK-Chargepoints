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
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// TODO: CreateCollection
// TODO: DeleteCollection

//
// Create
//

// Creates a new value in the key value store. If the key already exists then
// this call will return an error.
func (c *Collection) Create(key string, value interface{}) (*Item, error) {
	headers := map[string]string{"If-None-Match": `"*"`}
	item, err := c.innerPut(key, headers, value)
	if err != nil {
		if _, ok := err.(PreconditionFailedError); ok {
			err = AlreadyExistsError(key)
		}
	}
	return item, err
}

//
// Delete
//

// Unconditionally deletes the most recent version of the given item from
// the collection. This call will succeed even if the item didn't exist in the
// collection before this call.
func (c *Collection) Delete(key string) error {
	path := c.Name + "/" + key
	_, err := c.client.emptyReply("DELETE", path, nil, nil, 204)
	return err
}

// Unconditionally deletes all of the revisions of a object from the
// collection. This operation can not be undone.
func (c *Collection) Purge(key string) error {
	path := c.Name + "/" + key + "?purge=true"
	_, err := c.client.emptyReply("DELETE", path, nil, nil, 204)
	return err
}

//
// Get
//

// Get a key-value object from a Collection. The results will be stored
// in the value provided here. If value is non nil then the body of the
// results will be json decoded into the object given.
func (c *Collection) Get(key string, value interface{}) (*Item, error) {
	return c.GetRef(key, "", value)
}

// Gets a specific revision of an object. This works like Get() except that
// it takes the ref parameter which comes from either the 'Content-Location'
// header on a GET request, or the 'Location' header on a PUT request. If
// revision is an empty string then this fetches the most recent version
// [same as Get()]. If value is non nil then the results will be JSON
// decoded into the object given.
func (c *Collection) GetRef(
	key, ref string, value interface{},
) (*Item, error) {
	item := &Item{
		Collection: c,
		Key:        key,
	}

	// Get the path for the request and then query against Orchestrate.
	var path string
	if ref == "" {
		path = c.Name + "/" + key
	} else {
		path = c.Name + "/" + key + "/refs/" + ref
	}
	resp, err := c.client.jsonReply("GET", path, nil, 200, &item.Value)
	if err != nil {
		return nil, err
	}

	// Get the ref value.
	if ref == "" {
		loc := resp.Header.Get("Content-Location")
		if i := strings.LastIndex(loc, "/"); i == -1 {
			return nil, errors.New("Missing Content-Location header.")
		} else {
			item.Ref = loc[i+1 : len(loc)]
		}
	} else {
		item.Ref = ref
	}

	// If the user provided a value then decode into that value.
	if value != nil {
		return item, item.Unmarshal(value)
	}

	// Success!
	return item, nil
}

//
// History
//

// Used to query the history of a given object.
type HistoryQuery struct {
	// The number of items that should be returned per call to Orchestrate.
	// If unset this will be 10, and the maximum is 100.
	Limit int

	// The offset that the queries should start at. This allows a listing
	// to skip the first N elements.
	Offset int64

	// If this is true then the values will be returned with the objects.
	// Leaving this false will like result in faster queries but will
	// cause calls to Unmarshal to fail. The default for this is false.
	Values bool
}

// Returns the history of an object as an iterator. Note that this iterator
// will return the items most recent first. If opts is nil then the default
// options will be used which is no values and limit to 10 items per query
// (though the iterator will handle fetching the next block for you.)
func (c *Collection) History(key string, opts *HistoryQuery) *Iterator {
	var path string

	// Build a query from the user provided values.
	if opts != nil {
		queryVariables := make(url.Values, 10)
		if opts.Limit != 0 {
			queryVariables.Add("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset != 0 {
			queryVariables.Add("offset", strconv.FormatInt(opts.Offset, 10))
		}
		if opts.Values {
			queryVariables.Add("values", "true")
		}

		path = c.Name + "/" + key + "/refs?" + queryVariables.Encode()
	} else {
		path = c.Name + "/" + key + "/refs?"
	}

	return &Iterator{
		client:         c.client,
		iteratingItems: true,
		next:           path,
	}
}

//
// List
//

// Provides listing query parameters to a call to List().
type ListQuery struct {
	// The number of results to return per call to Orchestrate. The default
	// if this is not set is to return 10 at a time, the maximum that can be
	// returned is 100.
	Limit int

	// The key that should be used as the starting key for pagination. This
	// key and all that follow it (up until Limit) will be returned in the
	// call.
	StartKey string

	// Like StartKey except that this key will NOT be included in the listing.
	AfterKey string

	// All keys before this key will be included in the listing.
	BeforeKey string

	// All keys before this key, as well as this key will be included in the
	// listing.
	EndKey string
}

// Sets up a list query. Note that the actual query will not be performed
// until Next() is called on the Iterator returned.
func (c *Collection) List(query *ListQuery) *Iterator {
	path := c.Name

	// Build a query from the user provided values.
	if query != nil {
		queryVariables := make(url.Values, 10)
		if query.Limit != 0 {
			queryVariables.Add("limit", strconv.Itoa(query.Limit))
		}
		if query.AfterKey != "" {
			queryVariables.Add("afterKey", query.AfterKey)
		}
		if query.BeforeKey != "" {
			queryVariables.Add("beforeKey", query.BeforeKey)
		}
		if query.EndKey != "" {
			queryVariables.Add("endKey", query.EndKey)
		}
		if query.StartKey != "" {
			queryVariables.Add("startKey", query.StartKey)
		}

		path = c.Name + "?" + queryVariables.Encode()
	}

	return &Iterator{
		client:         c.client,
		iteratingItems: true,
		next:           path,
	}
}

//
// Search
//

// Provides optional searching parameters to a cal to Search()
type SearchQuery struct {
	// The number of results to return per call to Orchestrate. The default
	// if this is not set is to return 10 at a time, the maximum that can be
	// returned is 100.
	Limit int

	// The offset into the results that should be returned. This allows a
	// Search operation to skip the first x results.
	Offset int64

	// Determine sort ordering on the search. If this is an empty string
	// then the ordering will be based on scores.
	// TODO: Link to documentation on this field.
	Sort string
}

// Sets up a search query. If opts is nil then the default options will be
// used to query. Note that the actual query will not be performed until
// Next() is called on the Iterator returned. See the Iteration Example
// for details on how this works.
//
// For information on the query syntax see the documentation at:
//   <a href="http://orchestrate.io/docs/search">http://orchestrate.io/docs/search</a>
//
// Or alternatively the Lucene query syntax page at:
//   <a href="http://lucene.apache.org/core/4_5_1/queryparser/org/apache/lucene/queryparser/classic/package-summary.html#Overview">http://lucene.apache.org/core/4_5_1/queryparser/org/apache/lucene/queryparser/classic/package-summary.html#Overview</a>
func (c *Collection) Search(query string, opts *SearchQuery) *Iterator {
	queryVariables := make(url.Values, 10)
	queryVariables.Add("query", query)

	// Build a query from the user provided values.
	if opts != nil {
		if opts.Limit != 0 {
			queryVariables.Add("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset != 0 {
			queryVariables.Add("offset", strconv.FormatInt(opts.Offset, 10))
		}
		if opts.Sort != "" {
			queryVariables.Add("sort", opts.Sort)
		}

	}

	return &Iterator{
		client:         c.client,
		iteratingItems: true,
		next:           c.Name + "?" + queryVariables.Encode(),
	}
}

//
// Update (PUT)
//

// Updates a given key in the collection. If the object does not exist then
// this call will create it and returned a CreatedError as its error type.
func (c *Collection) Update(key string, value interface{}) (*Item, error) {
	return c.innerPut(key, nil, value)
}

//
// Private
//

// This is the inner Put implementation for Create(), Update() and
// Item.Update().
func (c *Collection) innerPut(
	key string, headers map[string]string, value interface{},
) (*Item, error) {
	item := &Item{
		Collection: c,
		Key:        key,
	}

	// Encode the json message into a raw value that we can return to the
	// client if necessary.
	if rawMsg, err := json.Marshal(value); err != nil {
		return nil, err
	} else {
		item.Value = json.RawMessage(rawMsg)
	}

	// Make the actual PUT call.
	path := c.Name + "/" + key
	resp, err := c.client.emptyReply("PUT", path, headers,
		bytes.NewBuffer([]byte(item.Value)), 201)
	if err != nil {
		return nil, err
	}

	// Get the ref from the returned strings.
	loc := resp.Header.Get("Location")
	if i := strings.LastIndex(loc, "/"); i == -1 {
		return nil, errors.New("Missing Location header.")
	} else {
		item.Ref = loc[i+1 : len(loc)]
	}

	return item, err
}
