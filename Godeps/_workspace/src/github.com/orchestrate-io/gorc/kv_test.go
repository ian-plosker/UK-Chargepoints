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
	"testing"
	"testing/quick"
)

func TestKVHasNext(t *testing.T) {
	f := func(results *KVResults) bool {
		return !(results.Next == "" && results.HasNext())
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestKVTrailingGetUri(t *testing.T) {
	f := func(path *Path) bool {
		if path.Ref == "" {
			return path.trailingGetURI() == path.Collection+"/"+path.Key
		}
		return path.trailingGetURI() == path.Collection+"/"+path.Key+"/refs/"+path.Ref
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestKVTrailingPutUri(t *testing.T) {
	f := func(path *Path) bool {
		return path.trailingPutURI() == path.Collection+"/"+path.Key
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
