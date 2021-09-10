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

// WARNING: This file has automatically been generated on Tue, 08 May 2018 08:48:40 EDT.
// By https://git.io/c-for-go. DO NOT EDIT.

package ovsh

const (
	// DatapathFamily as defined in ovsh/openvswitch.h:42
	DatapathFamily = "ovs_datapath"
	// DatapathMcgroup as defined in ovsh/openvswitch.h:43
	DatapathMcgroup = "ovs_datapath"
	// DatapathVersion as defined in ovsh/openvswitch.h:49
	DatapathVersion = 2
	// DpVerFeatures as defined in ovsh/openvswitch.h:52
	DpVerFeatures = 2
	// DpAttrMax as defined in ovsh/openvswitch.h:92
	DpAttrMax = (__DpAttrMax - 1)
	// DpFUnaligned as defined in ovsh/openvswitch.h:121
	DpFUnaligned = (1 << 0)
	// DpFVportPids as defined in ovsh/openvswitch.h:124
	DpFVportPids = (1 << 1)
	// PacketFamily as defined in ovsh/openvswitch.h:131
	PacketFamily = "ovs_packet"
	// PacketVersion as defined in ovsh/openvswitch.h:132
	PacketVersion = 0x1
	// PacketAttrMax as defined in ovsh/openvswitch.h:194
	PacketAttrMax = (__PacketAttrMax - 1)
	// VportFamily as defined in ovsh/openvswitch.h:198
	VportFamily = "ovs_vport"
	// VportMcgroup as defined in ovsh/openvswitch.h:199
	VportMcgroup = "ovs_vport"
	// VportVersion as defined in ovsh/openvswitch.h:200
	VportVersion = 0x1
	// VportTypeMax as defined in ovsh/openvswitch.h:220
	VportTypeMax = (__VportTypeMax - 1)
	// VportAttrMax as defined in ovsh/openvswitch.h:266
	VportAttrMax = (__VportAttrMax - 1)
	// VxlanExtMax as defined in ovsh/openvswitch.h:274
	VxlanExtMax = (__VxlanExtMax - 1)
	// TunnelAttrMax as defined in ovsh/openvswitch.h:286
	TunnelAttrMax = (__TunnelAttrMax - 1)
	// FlowFamily as defined in ovsh/openvswitch.h:290
	FlowFamily = "ovs_flow"
	// FlowMcgroup as defined in ovsh/openvswitch.h:291
	FlowMcgroup = "ovs_flow"
	// FlowVersion as defined in ovsh/openvswitch.h:292
	FlowVersion = 0x1
	// KeyAttrMax as defined in ovsh/openvswitch.h:347
	KeyAttrMax = (__KeyAttrMax - 1)
	// TunnelKeyAttrMax as defined in ovsh/openvswitch.h:370
	TunnelKeyAttrMax = (__TunnelKeyAttrMax - 1)
	// FragTypeMax as defined in ovsh/openvswitch.h:388
	FragTypeMax = (__FragTypeMax - 1)
	// CtLabelsLen32 as defined in ovsh/openvswitch.h:457
	CtLabelsLen32 = 4
	// CsFNew as defined in ovsh/openvswitch.h:467
	CsFNew = 0x01
	// CsFEstablished as defined in ovsh/openvswitch.h:468
	CsFEstablished = 0x02
	// CsFRelated as defined in ovsh/openvswitch.h:469
	CsFRelated = 0x04
	// CsFReplyDir as defined in ovsh/openvswitch.h:471
	CsFReplyDir = 0x08
	// CsFInvalid as defined in ovsh/openvswitch.h:472
	CsFInvalid = 0x10
	// CsFTracked as defined in ovsh/openvswitch.h:473
	CsFTracked = 0x20
	// CsFSrcNat as defined in ovsh/openvswitch.h:474
	CsFSrcNat = 0x40
	// CsFDstNat as defined in ovsh/openvswitch.h:477
	CsFDstNat = 0x80
	// CsFNatMask as defined in ovsh/openvswitch.h:481
	CsFNatMask = (CsFSrcNat | CsFDstNat)
	// NshKeyAttrMax as defined in ovsh/openvswitch.h:507
	NshKeyAttrMax = (__NshKeyAttrMax - 1)
	// FlowAttrMax as defined in ovsh/openvswitch.h:582
	FlowAttrMax = (__FlowAttrMax - 1)
	// UfidFOmitKey as defined in ovsh/openvswitch.h:590
	UfidFOmitKey = (1 << 0)
	// UfidFOmitMask as defined in ovsh/openvswitch.h:591
	UfidFOmitMask = (1 << 1)
	// UfidFOmitActions as defined in ovsh/openvswitch.h:592
	UfidFOmitActions = (1 << 2)
	// SampleAttrMax as defined in ovsh/openvswitch.h:617
	SampleAttrMax = (__SampleAttrMax - 1)
	// UserspaceAttrMax as defined in ovsh/openvswitch.h:650
	UserspaceAttrMax = (__UserspaceAttrMax - 1)
	// CtAttrMax as defined in ovsh/openvswitch.h:752
	CtAttrMax = (__CtAttrMax - 1)
	// NatAttrMax as defined in ovsh/openvswitch.h:790
	NatAttrMax = (__NatAttrMax - 1)
	// ActionAttrMax as defined in ovsh/openvswitch.h:887
	ActionAttrMax = (__ActionAttrMax - 1)
	// MeterFamily as defined in ovsh/openvswitch.h:890
	MeterFamily = "ovs_meter"
	// MeterMcgroup as defined in ovsh/openvswitch.h:891
	MeterMcgroup = "ovs_meter"
	// MeterVersion as defined in ovsh/openvswitch.h:892
	MeterVersion = 0x1
	// MeterAttrMax as defined in ovsh/openvswitch.h:919
	MeterAttrMax = (__MeterAttrMax - 1)
	// BandAttrMax as defined in ovsh/openvswitch.h:930
	BandAttrMax = (__BandAttrMax - 1)
	// MeterBandTypeMax as defined in ovsh/openvswitch.h:938
	MeterBandTypeMax = (__MeterBandTypeMax - 1)
)

// ovsDatapathCmd as declared in ovsh/openvswitch.h:54
//type ovsDatapathCmd int32

// ovsDatapathCmd enumeration from ovsh/openvswitch.h:54
const (
	DpCmdUnspec = iota
	DpCmdNew    = 1
	DpCmdDel    = 2
	DpCmdGet    = 3
	DpCmdSet    = 4
)

// ovsDatapathAttr as declared in ovsh/openvswitch.h:81
//type ovsDatapathAttr int32

// ovsDatapathAttr enumeration from ovsh/openvswitch.h:81
const (
	DpAttrUnspec        = iota
	DpAttrName          = 1
	DpAttrUpcallPid     = 2
	DpAttrStats         = 3
	DpAttrMegaflowStats = 4
	DpAttrUserFeatures  = 5
	DpAttrPad           = 6
	__DpAttrMax         = 7
)

// ovsPacketCmd as declared in ovsh/openvswitch.h:134
//type ovsPacketCmd int32

// ovsPacketCmd enumeration from ovsh/openvswitch.h:134
const (
	PacketCmdUnspec  = iota
	PacketCmdMiss    = 1
	PacketCmdAction  = 2
	PacketCmdExecute = 3
)

// ovsPacketAttr as declared in ovsh/openvswitch.h:177
//type ovsPacketAttr int32

// ovsPacketAttr enumeration from ovsh/openvswitch.h:177
const (
	PacketAttrUnspec       = iota
	PacketAttrPacket       = 1
	PacketAttrKey          = 2
	PacketAttrActions      = 3
	PacketAttrUserdata     = 4
	PacketAttrEgressTunKey = 5
	PacketAttrUnused1      = 6
	PacketAttrUnused2      = 7
	PacketAttrProbe        = 8
	PacketAttrMru          = 9
	PacketAttrLen          = 10
	__PacketAttrMax        = 11
)

// ovsVportCmd as declared in ovsh/openvswitch.h:202
//type ovsVportCmd int32

// ovsVportCmd enumeration from ovsh/openvswitch.h:202
const (
	VportCmdUnspec = iota
	VportCmdNew    = 1
	VportCmdDel    = 2
	VportCmdGet    = 3
	VportCmdSet    = 4
)

// ovsVportType as declared in ovsh/openvswitch.h:210
//type ovsVportType int32

// ovsVportType enumeration from ovsh/openvswitch.h:210
const (
	VportTypeUnspec   = iota
	VportTypeNetdev   = 1
	VportTypeInternal = 2
	VportTypeGre      = 3
	VportTypeVxlan    = 4
	VportTypeGeneve   = 5
	__VportTypeMax    = 6
)

// ovsVportAttr as declared in ovsh/openvswitch.h:251
//type ovsVportAttr int32

// ovsVportAttr enumeration from ovsh/openvswitch.h:251
const (
	VportAttrUnspec    = iota
	VportAttrPortNo    = 1
	VportAttrType      = 2
	VportAttrName      = 3
	VportAttrOptions   = 4
	VportAttrUpcallPid = 5
	VportAttrStats     = 6
	VportAttrPad       = 7
	VportAttrIfindex   = 8
	VportAttrNetnsid   = 9
	__VportAttrMax     = 10
)

const (
	// VxlanExtUnspec as declared in ovsh/openvswitch.h:269
	VxlanExtUnspec = iota
	// VxlanExtGbp as declared in ovsh/openvswitch.h:270
	VxlanExtGbp = 1
	// __VxlanExtMax as declared in ovsh/openvswitch.h:271
	__VxlanExtMax = 2
)

const (
	// TunnelAttrUnspec as declared in ovsh/openvswitch.h:280
	TunnelAttrUnspec = iota
	// TunnelAttrDstPort as declared in ovsh/openvswitch.h:281
	TunnelAttrDstPort = 1
	// TunnelAttrExtension as declared in ovsh/openvswitch.h:282
	TunnelAttrExtension = 2
	// __TunnelAttrMax as declared in ovsh/openvswitch.h:283
	__TunnelAttrMax = 3
)

// ovsFlowCmd as declared in ovsh/openvswitch.h:294
//type ovsFlowCmd int32

// ovsFlowCmd enumeration from ovsh/openvswitch.h:294
const (
	FlowCmdUnspec = iota
	FlowCmdNew    = 1
	FlowCmdDel    = 2
	FlowCmdGet    = 3
	FlowCmdSet    = 4
)

// ovsKeyAttr as declared in ovsh/openvswitch.h:307
//type ovsKeyAttr int32

// ovsKeyAttr enumeration from ovsh/openvswitch.h:307
const (
	KeyAttrUnspec          = iota
	KeyAttrEncap           = 1
	KeyAttrPriority        = 2
	KeyAttrInPort          = 3
	KeyAttrEthernet        = 4
	KeyAttrVlan            = 5
	KeyAttrEthertype       = 6
	KeyAttrIpv4            = 7
	KeyAttrIpv6            = 8
	KeyAttrTcp             = 9
	KeyAttrUdp             = 10
	KeyAttrIcmp            = 11
	KeyAttrIcmpv6          = 12
	KeyAttrArp             = 13
	KeyAttrNd              = 14
	KeyAttrSkbMark         = 15
	KeyAttrTunnel          = 16
	KeyAttrSctp            = 17
	KeyAttrTcpFlags        = 18
	KeyAttrDpHash          = 19
	KeyAttrRecircId        = 20
	KeyAttrMpls            = 21
	KeyAttrCtState         = 22
	KeyAttrCtZone          = 23
	KeyAttrCtMark          = 24
	KeyAttrCtLabels        = 25
	KeyAttrCtOrigTupleIpv4 = 26
	KeyAttrCtOrigTupleIpv6 = 27
	KeyAttrNsh             = 28
	__KeyAttrMax           = 29
)

// ovsTunnelKeyAttr as declared in ovsh/openvswitch.h:349
//type ovsTunnelKeyAttr int32

// ovsTunnelKeyAttr enumeration from ovsh/openvswitch.h:349
const (
	TunnelKeyAttrId           = iota
	TunnelKeyAttrIpv4Src      = 1
	TunnelKeyAttrIpv4Dst      = 2
	TunnelKeyAttrTos          = 3
	TunnelKeyAttrTtl          = 4
	TunnelKeyAttrDontFragment = 5
	TunnelKeyAttrCsum         = 6
	TunnelKeyAttrOam          = 7
	TunnelKeyAttrGeneveOpts   = 8
	TunnelKeyAttrTpSrc        = 9
	TunnelKeyAttrTpDst        = 10
	TunnelKeyAttrVxlanOpts    = 11
	TunnelKeyAttrIpv6Src      = 12
	TunnelKeyAttrIpv6Dst      = 13
	TunnelKeyAttrPad          = 14
	TunnelKeyAttrErspanOpts   = 15
	__TunnelKeyAttrMax        = 16
)

// ovsFragType as declared in ovsh/openvswitch.h:381
//type ovsFragType int32

// ovsFragType enumeration from ovsh/openvswitch.h:381
const (
	FragTypeNone  = iota
	FragTypeFirst = 1
	FragTypeLater = 2
	__FragTypeMax = 3
)

// ovsNshKeyAttr as declared in ovsh/openvswitch.h:499
//type ovsNshKeyAttr int32

// ovsNshKeyAttr enumeration from ovsh/openvswitch.h:499
const (
	NshKeyAttrUnspec = iota
	NshKeyAttrBase   = 1
	NshKeyAttrMd1    = 2
	NshKeyAttrMd2    = 3
	__NshKeyAttrMax  = 4
)

// ovsFlowAttr as declared in ovsh/openvswitch.h:565
//type ovsFlowAttr int32

// ovsFlowAttr enumeration from ovsh/openvswitch.h:565
const (
	FlowAttrUnspec    = iota
	FlowAttrKey       = 1
	FlowAttrActions   = 2
	FlowAttrStats     = 3
	FlowAttrTcpFlags  = 4
	FlowAttrUsed      = 5
	FlowAttrClear     = 6
	FlowAttrMask      = 7
	FlowAttrProbe     = 8
	FlowAttrUfid      = 9
	FlowAttrUfidFlags = 10
	FlowAttrPad       = 11
	__FlowAttrMax     = 12
)

// ovsSampleAttr as declared in ovsh/openvswitch.h:606
//type ovsSampleAttr int32

// ovsSampleAttr enumeration from ovsh/openvswitch.h:606
const (
	SampleAttrUnspec      = iota
	SampleAttrProbability = 1
	SampleAttrActions     = 2
	__SampleAttrMax       = 3
)

// ovsUserspaceAttr as declared in ovsh/openvswitch.h:640
//type ovsUserspaceAttr int32

// ovsUserspaceAttr enumeration from ovsh/openvswitch.h:640
const (
	UserspaceAttrUnspec        = iota
	UserspaceAttrPid           = 1
	UserspaceAttrUserdata      = 2
	UserspaceAttrEgressTunPort = 3
	UserspaceAttrActions       = 4
	__UserspaceAttrMax         = 5
)

// ovsHashAlg as declared in ovsh/openvswitch.h:692
//type ovsHashAlg int32

// ovsHashAlg enumeration from ovsh/openvswitch.h:692
const (
	HashAlgL4 = iota
)

// ovsCtAttr as declared in ovsh/openvswitch.h:738
//type ovsCtAttr int32

// ovsCtAttr enumeration from ovsh/openvswitch.h:738
const (
	CtAttrUnspec      = iota
	CtAttrCommit      = 1
	CtAttrZone        = 2
	CtAttrMark        = 3
	CtAttrLabels      = 4
	CtAttrHelper      = 5
	CtAttrNat         = 6
	CtAttrForceCommit = 7
	CtAttrEventmask   = 8
	__CtAttrMax       = 9
)

// ovsNatAttr as declared in ovsh/openvswitch.h:776
//type ovsNatAttr int32

// ovsNatAttr enumeration from ovsh/openvswitch.h:776
const (
	NatAttrUnspec      = iota
	NatAttrSrc         = 1
	NatAttrDst         = 2
	NatAttrIpMin       = 3
	NatAttrIpMax       = 4
	NatAttrProtoMin    = 5
	NatAttrProtoMax    = 6
	NatAttrPersistent  = 7
	NatAttrProtoHash   = 8
	NatAttrProtoRandom = 9
	__NatAttrMax       = 10
)

// ovsActionAttr as declared in ovsh/openvswitch.h:852
//type ovsActionAttr int32

// ovsActionAttr enumeration from ovsh/openvswitch.h:852
const (
	ActionAttrUnspec    = iota
	ActionAttrOutput    = 1
	ActionAttrUserspace = 2
	ActionAttrSet       = 3
	ActionAttrPushVlan  = 4
	ActionAttrPopVlan   = 5
	ActionAttrSample    = 6
	ActionAttrRecirc    = 7
	ActionAttrHash      = 8
	ActionAttrPushMpls  = 9
	ActionAttrPopMpls   = 10
	ActionAttrSetMasked = 11
	ActionAttrCt        = 12
	ActionAttrTrunc     = 13
	ActionAttrPushEth   = 14
	ActionAttrPopEth    = 15
	ActionAttrCtClear   = 16
	ActionAttrPushNsh   = 17
	ActionAttrPopNsh    = 18
	ActionAttrMeter     = 19
	__ActionAttrMax     = 20
)

// ovsMeterCmd as declared in ovsh/openvswitch.h:894
//type ovsMeterCmd int32

// ovsMeterCmd enumeration from ovsh/openvswitch.h:894
const (
	MeterCmdUnspec   = iota
	MeterCmdFeatures = 1
	MeterCmdSet      = 2
	MeterCmdDel      = 3
	MeterCmdGet      = 4
)

// ovsMeterAttr as declared in ovsh/openvswitch.h:902
//type ovsMeterAttr int32

// ovsMeterAttr enumeration from ovsh/openvswitch.h:902
const (
	MeterAttrUnspec    = iota
	MeterAttrId        = 1
	MeterAttrKbps      = 2
	MeterAttrStats     = 3
	MeterAttrBands     = 4
	MeterAttrUsed      = 5
	MeterAttrClear     = 6
	MeterAttrMaxMeters = 7
	MeterAttrMaxBands  = 8
	MeterAttrPad       = 9
	__MeterAttrMax     = 10
)

// ovsBandAttr as declared in ovsh/openvswitch.h:921
//type ovsBandAttr int32

// ovsBandAttr enumeration from ovsh/openvswitch.h:921
const (
	BandAttrUnspec = iota
	BandAttrType   = 1
	BandAttrRate   = 2
	BandAttrBurst  = 3
	BandAttrStats  = 4
	__BandAttrMax  = 5
)

// ovsMeterBandType as declared in ovsh/openvswitch.h:932
//type ovsMeterBandType int32

// ovsMeterBandType enumeration from ovsh/openvswitch.h:932
const (
	MeterBandTypeUnspec = iota
	MeterBandTypeDrop   = 1
	__MeterBandTypeMax  = 2
)
