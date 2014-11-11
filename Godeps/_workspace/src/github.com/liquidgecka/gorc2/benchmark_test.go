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

	"github.com/orchestrate-io/dvr"
)

// Skips the benchmark if dvr is not in PassThrough mode.
func SkipIfNotPassingThrough(b *testing.B) {
	if !dvr.IsPassingThrough() {
		b.Logf("Benchmarks can only be run in passthrough mode.")
		b.Logf("Please use -dvr.passthrough")
		b.SkipNow()
	}
}

func benchmarkPing(b *testing.B, routines int) {
	b.N = 25 * routines
	SkipIfNotPassingThrough(b)
	client := NewClient(testAuthToken)

	locker := Locker{}
	in := make(chan bool, 100)
	for i := 0; i < routines; i++ {
		locker.Add(1)
		go func() {
			defer locker.Done()
			for v := range in {
				if v && locker.IsSet() {
					continue
				}
				locker.Set(client.Ping())
			}
		}()
	}

	// Start the benchmark and feed the channel.
	b.SetBytes(int64(b.N) * 1000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		in <- true
	}
	close(in)

	// Wait for the test to finish.
	locker.Wait()
	if locker.Err != nil {
		b.Fatalf("A request failed: %s", locker.Err)
	}
}

func BenchmarkPing_Serial(b *testing.B) {
	b.SetBytes(int64(b.N) * 1000000)
	SkipIfNotPassingThrough(b)
	client := NewClient(testAuthToken)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := client.Ping()
		if err != nil {
			b.Fatalf("Error from Orchestrate: %s", err)
		}
	}
}

func BenchmarkPing_2Goroutines(b *testing.B) {
	benchmarkPing(b, 2)
}

func BenchmarkPing_5Goroutines(b *testing.B) {
	benchmarkPing(b, 5)
}

func BenchmarkPing_10Goroutines(b *testing.B) {
	benchmarkPing(b, 10)
}

func BenchmarkPing_25Goroutines(b *testing.B) {
	benchmarkPing(b, 25)
}

func BenchmarkPing_50Goroutines(b *testing.B) {
	benchmarkPing(b, 50)
}
