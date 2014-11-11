gorc2
=====

A golang library for Orchestrate.io.

## Installation

```bash
go get githup.com/orchestrate-io/gorc2
```

## Usage

The [godocs](http://godoc.org/github.com/orchestrate-io/gorc2) for this project
contain examples showing most usage. In order to start you must sign up for a
free Orchestrate account on the [Dashboard](https://dashboard.orchestrate.io).
Once done you should create an Application, then use the authorization token
listed in the examples.

This library has no dependencies outside of the core golang libraries.

## Testing

[![Continuous Integration](https://secure.travis-ci.org/orchestrate-io/gorc2.svg?branch=master)](http://travis-ci.org/orchestrate-io/gorc2)
[![Documentation](http://godoc.org/github.com/orchestrate-io/gorc2?status.png)](http://godoc.org/github.com/orchestrate-io/gorc2)
[![Coverage](https://img.shields.io/coveralls/orchestrate-io/gorc2.svg)](https://coveralls.io/r/orchestrate-io/gorc2)

In order to test the gorc2 library you need two supporting libraries, they can
be installed by running:
```bash
go get githup.com/liquidgecka/testlib
go get githup.com/orchestrate-io/dvr
```

The tests can be run via the traditional golang unit testing framework, and
running them should not require network access, or even an account with
Orchestrate. However, if you are editing tests you might find that running
the tests panics. This is due to the way we use the
[DVR](http://github.com/orchestrate-io/dvr) library. Typically requests are
replayed during tests, but if you added a new call to Orchestrate it will need
to be added to the dvr archive. To do this run the tests with the -dvr.record
and -auth_token flags.

```bash
# Normal test run which doesn't talk to the network.
go test -v .

# Recording run (replace the token with your own).
go test -v . -dvr.record -auth_token=aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
```

In recording mode the results of the queries (less your auth token) are saved
into *testdata/dvr.archive* for use with later queries. When submitting a pull
request this step will likely have to have been done to get Travis to work.

## Contribution

Pull requests will be reviewed and accepted via github.com, and issues will be
worked on as time permits. New feature requests should be filed as an issue
against the github repo.

## License (Apache 2)

Copyright 2014 Orchestrate, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
