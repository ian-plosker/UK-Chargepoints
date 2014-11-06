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
	"strings"
)

// Holds results returned from a Graph query.
type GraphResults struct {
	Count   uint64        `json:"count"`
	Results []GraphResult `json:"results"`
}

// An individual graph result.
type GraphResult struct {
	Path     Path            `json:"path"`
	RawValue json.RawMessage `json:"value"`
}

// Get all related key/value objects by collection-key and a list of relations.
func (c *Client) GetRelations(collection, key string, hops []string) (*GraphResults, error) {
	relationsPath := strings.Join(hops, "/")

	trailingUri := collection + "/" + key + "/relations/" + relationsPath
	resp, err := c.doRequest("GET", trailingUri, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If the response was an error we return an OrchestrateError object.
	if resp.StatusCode != 200 {
		return nil, newError(resp)
	}

	// Read the body into the results.
	decoder := json.NewDecoder(resp.Body)
	result := new(GraphResults)
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}

// Create a relationship of a specified type between two collection-keys.
func (c *Client) PutRelation(sourceCollection, sourceKey, kind, sinkCollection, sinkKey string) error {
	trailingUri := sourceCollection + "/" + sourceKey + "/relation/" + kind + "/" + sinkCollection + "/" + sinkKey
	resp, err := c.doRequest("PUT", trailingUri, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If the response was an error we return an OrchestrateError object which
	// reads the body.
	if resp.StatusCode != 204 {
		return newError(resp)
	}

	// Otherwise we need to read it ourselves.
	io.Copy(ioutil.Discard, resp.Body)

	return nil
}

// Create a relationship of a specified type between two collection-keys.
func (c *Client) DeleteRelation(sourceCollection string, sourceKey string, kind string, sinkCollection string, sinkKey string) error {
	trailingUri := sourceCollection + "/" + sourceKey + "/relation/" + kind + "/" + sinkCollection + "/" + sinkKey + "?purge=true"
	resp, err := c.doRequest("DELETE", trailingUri, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If the response was an error we return an OrchestrateError object
	// which reads the body.
	if resp.StatusCode != 204 {
		return newError(resp)
	}

	// Otherwise we need to read it ourselves.
	io.Copy(ioutil.Discard, resp.Body)

	return nil
}

// Marshall the value of a GraphResult into the provided object.
func (r *GraphResult) Value(value interface{}) error {
	return json.Unmarshal(r.RawValue, value)
}
