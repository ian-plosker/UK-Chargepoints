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
	"encoding/json"
	"fmt"
	"net/http"
)

// Creates a new UnknownError from a given http.Response object.
func newError(resp *http.Response) error {
	switch resp.StatusCode {
	case 404:
		return NotFoundError("404: Not found.")
	case 412:
		return PreconditionFailedError("412: Precondition failed.")
	case 419:
		return RateLimitedError("Request rate limited.")
	}
	oe := &UnknownError{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(oe); err != nil {
		oe.Message = err.Error()
		return oe
	}

	return oe
}

// AlreadyExistsError (412 for Create)

// A error type that is returned when an item already exists which prevents
// a creation.
type AlreadyExistsError string

func (a AlreadyExistsError) Error() string {
	return fmt.Sprintf("An item with the key %s already exists.", string(a))
}

// NotMostRecentError (412 on Update/Delete)

// The error object returned if a Conditional*() call fails due to the item
// not being the most recent ref.
type NotMostRecentError string

func (a NotMostRecentError) Error() string {
	return fmt.Sprintf("%s was not the most recent ref.", string(a))
}

// NotFoundError (404)

// An error thrown when an item is not found.
type NotFoundError string

func (n NotFoundError) Error() string {
	return string(n)
}

// PreconditionFailedError (412)

// An error type returned when a 412 is returned from Orchestrate.
type PreconditionFailedError string

func (p PreconditionFailedError) Error() string {
	return string(p)
}

// RateLimitedError (419)

// An error type returned when a 412 is returned from Orchestrate.
type RateLimitedError string

func (p RateLimitedError) Error() string {
	return string(p)
}

// UnknownError

// An implementation of 'error' that exposes all the orchestrate specific
// error details.
type UnknownError struct {
	// The status string returned from the HTTP call.
	Status string `json:"-"`

	// The status, as an integer, returned from the HTTP call.
	StatusCode int `json:"-"`

	// The Orchestrate specific message representing the error.
	Message string `json:"message"`
}

// Convert the error to a meaningful string.
func (e UnknownError) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.Status, e.StatusCode, e.Message)
}
