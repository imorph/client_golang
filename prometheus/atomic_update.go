// Copyright 2014 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import (
	"math"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
)

// // backoffFuncType defines the signature for the backoff functions.
// type backoffFuncType func(attempts int, backoff *time.Duration)

// atomicUpdateFloat atomically updates the float64 value pointed to by bits
// using the provided updateFunc. It uses a backoff strategy that adapts to the CPU architecture.
func atomicUpdateFloat(bits *uint64, updateFunc func(float64) float64) {
	var backoff time.Duration
	var attempts int
	var calculateBackoff func(_ int, _ time.Duration) time.Duration

	if runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" {
		calculateBackoff = calculateLinearBackoff
	} else {
		calculateBackoff = calculateExpBackoff
	}

	for {
		loadedBits := atomic.LoadUint64(bits)
		oldFloat := math.Float64frombits(loadedBits)
		newFloat := updateFunc(oldFloat)
		newBits := math.Float64bits(newFloat)

		if atomic.CompareAndSwapUint64(bits, loadedBits, newBits) {
			break // Successful update
		} else {
			attempts++
			backoff = calculateBackoff(attempts, backoff)

			// Apply jitter to the backoff duration
			minSleep := backoff / 2
			maxSleep := backoff
			sleepDuration := minSleep + time.Duration(rand.Int63n(int64(maxSleep-minSleep)))
			time.Sleep(sleepDuration)
		}
	}
}

// calculateBackoff implements linear backoff
func calculateLinearBackoff(attempts int, _ time.Duration) time.Duration {
	const (
		baseBackoff = 1 * time.Millisecond
		maxBackoff  = 320 * time.Millisecond
	)

	backoff := baseBackoff * time.Duration(attempts)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}

// calculateBackoff implements exponential backoff with jitter for non-ARM architectures.
func calculateExpBackoff(_ int, previousBackoff time.Duration) time.Duration {
	const (
		initialBackoff = 10 * time.Millisecond
		maxBackoff     = 320 * time.Millisecond
	)

	if previousBackoff == 0 {
		return initialBackoff
	}

	backoff := previousBackoff * 2
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}
