// Copyright 2017 DigitalOcean.
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

//go:build !linux

package ovsnl

import (
	"errors"
)

// NewZoneMarkAggregator creates a new aggregator.
// On non-Linux platforms, this returns an error.
func NewZoneMarkAggregator() (*ZoneMarkAggregator, error) {
	return nil, errors.ErrUnsupported
}

// Start subscribes to conntrack events and maintains counts.
// On non-Linux platforms, this returns an error.
func (a *ZoneMarkAggregator) Start() error {
	return errors.ErrUnsupported
}

// Stop cancels listening.
func (a *ZoneMarkAggregator) Stop() {
	// No-op on non-Linux platforms
}

// Snapshot returns a safe copy of counts.
// On non-Linux platforms, this returns an empty map.
func (a *ZoneMarkAggregator) Snapshot() map[ZoneMarkKey]int {
	return make(map[ZoneMarkKey]int)
}
