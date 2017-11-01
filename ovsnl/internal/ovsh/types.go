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

//+build ignore

package ovsh

// #include "openvswitch.h"
import "C"

type Header C.struct_ovs_header

type DPStats C.struct_ovs_dp_stats

type DPMegaflowStats C.struct_ovs_dp_megaflow_stats

type VportStats C.struct_ovs_vport_stats

type FlowStats C.struct_ovs_flow_stats
