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
)

func TestAlreadyExistsError_Error(t *testing.T) {
	e := AlreadyExistsError("test")
	if e.Error() != "An item with the key test already exists." {
		t.Fatalf("Bad error message: %s", e.Error())
	}
}

func TestNotMostRecentError_Error(t *testing.T) {
	e := NotMostRecentError("test")
	if e.Error() != "test was not the most recent ref." {
		t.Fatalf("Bad error message: %s", e.Error())
	}
}

func TestNotFoundError_Error(t *testing.T) {
	e := NotFoundError("test")
	if e.Error() != "test" {
		t.Fatalf("Bad error message: %s", e.Error())
	}
}

func TestPreconditionFailedError_Error(t *testing.T) {
	e := PreconditionFailedError("test")
	if e.Error() != "test" {
		t.Fatalf("Bad error message: %s", e.Error())
	}
}

func TestRateLimitedError_Error(t *testing.T) {
	e := RateLimitedError("test")
	if e.Error() != "test" {
		t.Fatalf("Bad error message: %s", e.Error())
	}
}

func TestUnknownError_Error(t *testing.T) {
	e := UnknownError{
		Status: "status",
		StatusCode: 100,
		Message: "message",
	}
	if e.Error() != "status (100): message" {
		t.Fatalf("Bad error message: %s", e.Error())
	}
}
