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

// Created by cgo -godefs - DO NOT EDIT
// cgo -godefs types.go

package ovsh

type Header struct {
	Ifindex int32
}

type DPStats struct {
	Hit	uint64
	Missed	uint64
	Lost	uint64
	Flows	uint64
}

type DPMegaflowStats struct {
	Mask_hit	uint64
	Masks		uint32
	Pad0		uint32
	Pad1		uint64
	Pad2		uint64
}

type VportStats struct {
	Rx_packets	uint64
	Tx_packets	uint64
	Rx_bytes	uint64
	Tx_bytes	uint64
	Rx_errors	uint64
	Tx_errors	uint64
	Rx_dropped	uint64
	Tx_dropped	uint64
}

type FlowStats struct {
	Packets	uint64
	Bytes	uint64
}
