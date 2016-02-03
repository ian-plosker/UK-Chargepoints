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
	"strings"
	"strconv"
)

//
// GetLinks
//

// Wraps the options from GetLinks into a structure so more fields can be
// added later if necessary.
type GetLinksQuery struct {
	// The number of items that should be returned per call to Orchestrate.
	// If unset this will be 10, and the maximum is 100.
	Limit int
}

// Sets up an Iterator that will walk all of relations to the given key.
// The first kind is required, however optional other kinds may be added
// to traverse the graph further. If opts is null then default values will
// be used.
//
// For more information on how graphs work see this page:
//   <a href="http://orchestrate.io/docs/graph">http://orchestrate.io/docs/graph</a>
func (c *Collection) GetLinks(
	key string, opts *GetLinksQuery, kind string, kinds ...string,
) *Iterator {
	path := c.Name + "/" + key + "/relations/" + kind
	if len(kinds) > 0 {
		path = path + "/" + strings.Join(kinds, "/")
	}
	if opts != nil && opts.Limit != 0 {
		path = path + "?limit=" + strconv.Itoa(opts.Limit)
	}
	return &Iterator{
		client:         c.client,
		iteratingItems: true,
		next:           path,
	}
}


//
// Link
//

// Creates a graph link between two items.
// FIXME: Better documentation
func (c *Collection) Link(key, kind, toCollection, toKey string) error {
	path := fmt.Sprintf("%s/%s/relation/%s/%s/%s", c.Name, key, kind,
		toCollection, toKey)
	_, err := c.client.emptyReply("PUT", path, nil, nil, 204)
	return err
}

//
// Unlink
//

// Deletes a graph link between two items.
// FIXME: Better documentation
func (c *Collection) Unlink(key, kind, toCollection, toKey string) error {
	path := fmt.Sprintf("%s/%s/relation/%s/%s/%s?purge=true", c.Name, key,
		kind, toCollection, toKey)
	_, err := c.client.emptyReply("DELETE", path, nil, nil, 204)
	return err
}
