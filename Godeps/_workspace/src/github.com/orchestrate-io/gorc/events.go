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

package gorc

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
)

// Holds results returned from an Events query.
type EventResults struct {
	Count   uint64  `json:"count"`
	Results []Event `json:"results"`
}

// An individual event.
type Event struct {
	Ordinal   uint64          `json:"ordinal"`
	Timestamp uint64          `json:"timestamp"`
	RawValue  json.RawMessage `json:"value"`
}

// Get latest events of a particular type from specified collection-key pair.
func (c *Client) GetEvents(collection, key, kind string) (*EventResults, error) {
	trailingUri := collection + "/" + key + "/events/" + kind

	return c.doGetEvents(trailingUri)
}

// Get all events of a particular type from specified collection-key pair in a
// range.
func (c *Client) GetEventsInRange(collection, key, kind string, start int64, end int64) (*EventResults, error) {
	return c.GetEventsInRangeWithLimit(collection, key, kind, start, end, 10)
}

// Get all events of a particular type from a specified collection-key in a range with a limit
func (c *Client) GetEventsInRangeWithLimit(collection, key, kind string, start, end, limit int64) (*EventResults, error) {
	queryVariables := url.Values{
		"start": []string{strconv.FormatInt(start, 10)},
		"end":   []string{strconv.FormatInt(end, 10)},
		"limit": []string{strconv.FormatInt(limit, 10)},
	}

	trailingUri := collection + "/" + key + "/events/" + kind + "?" + queryVariables.Encode()

	return c.doGetEvents(trailingUri)
}

// Put an event of the specified type to provided collection-key pair.
func (c *Client) PutEvent(collection, key, kind string, value interface{}) error {
	reader, writer := io.Pipe()
	encoder := json.NewEncoder(writer)

	go func() { writer.CloseWithError(encoder.Encode(value)) }()
	return c.PutEventRaw(collection, key, kind, reader)
}

// Put an event of the specified type to provided collection-key pair.
func (c *Client) PutEventRaw(collection, key, kind string, value io.Reader) error {
	trailingUri := collection + "/" + key + "/events/" + kind

	return c.doPutEvent(trailingUri, value)

}

// Put an event of the specified type to provided collection-key pair and time.
func (c *Client) PutEventWithTime(collection, key, kind string, time int64, value interface{}) error {
	reader, writer := io.Pipe()
	encoder := json.NewEncoder(writer)

	go func() { writer.CloseWithError(encoder.Encode(value)) }()
	return c.PutEventWithTimeRaw(collection, key, kind, time, reader)
}

// Put an event of the specified type to provided collection-key pair and time.
func (c *Client) PutEventWithTimeRaw(collection, key, kind string, time int64, value io.Reader) error {
	queryVariables := url.Values{
		"timestamp": []string{strconv.FormatInt(time, 10)},
	}

	trailingUri := collection + "/" + key + "/events/" + kind + "?" + queryVariables.Encode()

	return c.doPutEvent(trailingUri, value)
}

// Execute event get.
func (c *Client) doGetEvents(trailingUri string) (*EventResults, error) {
	resp, err := c.doRequest("GET", trailingUri, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If the request ended in error then read the body into an
	// OrchestrateError object.
	if resp.StatusCode != 200 {
		return nil, newError(resp)
	}

	// Read the entire body into a new JSON object.
	decoder := json.NewDecoder(resp.Body)
	results := new(EventResults)
	if err = decoder.Decode(results); err != nil {
		return nil, err
	}

	return results, err
}

// Execute event put.
func (c *Client) doPutEvent(trailingUri string, value io.Reader) error {
	resp, err := c.doRequest("PUT", trailingUri, nil, value)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If the request ended in error then read the body into an
	// OrchestrateError object.
	if resp.StatusCode != 204 {
		return newError(resp)
	}

	// Read the body so the connection can be properly reused.
	io.Copy(ioutil.Discard, resp.Body)

	// Success
	return nil
}

// Marshall the value of an event into the provided object.
func (r *Event) Value(value interface{}) error {
	return json.Unmarshal(r.RawValue, value)
}
