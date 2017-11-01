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

// Package ovsh is an auto-generated package which contains constants and
// types used to access Open vSwitch information using generic netlink.
package ovsh

// Pull the latest openvswitch.h from the kernel for code generation.
//go:generate wget https://raw.githubusercontent.com/torvalds/linux/master/include/uapi/linux/openvswitch.h

// Generate Go source from C constants.
//go:generate c-for-go -out ../ -nocgo ovsh.yml

// Generate Go types for C types.
//go:generate sh -c "go tool cgo -godefs types.go > struct.go"

// Apply license headers to generated files.
//go:generate ../../../scripts/prependlicense.sh const.go
//go:generate ../../../scripts/prependlicense.sh struct.go

// Clean up build artifacts.
//go:generate rm -rf openvswitch.h _obj/
