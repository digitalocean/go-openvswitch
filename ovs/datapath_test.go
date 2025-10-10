// Copyright 2021 DigitalOcean.
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

package ovs

import (
	"errors"
	"testing"
)

const (
	validTest = iota
	handleError
	unknownCommand
)

// MockOvsCLI is mocking the "ovs-dpctl (Open vSwitch) 2.10.7" version
type MockOvsCLI struct {
	Version uint8
}

func (cli *MockOvsCLI) Exec(args ...string) ([]byte, error) {
	cmd := args[0]
	switch cmd {
	case "--version":
		return []byte("ovs-dpctl (Open vSwitch) 2.10.7"), nil
	case "dump-dps":
		return []byte("system@ovs-system\nsystem@dp-test"), nil
	case "add-dp":
		if args[1] == "ovs-system" {
			return []byte{}, errors.New("datapath already exists")

		}
		return []byte{}, nil
	case "del-dp":
		return []byte{}, nil
	case "ct-get-limits":
		if cli.Version < 10 {
			return []byte{}, errors.New("ovs-dpctl: unknown command 'ct-get-limits'; use --help for help")
		}
		out := "default limit=0\nzone=2,limit=1000000,count=0\nzone=3,limit=1000000,count=0"
		return []byte(out), nil
	case "ct-set-limits":
		if cli.Version < 10 {
			return []byte{}, errors.New("ovs-dpctl: unknown command 'ct-get-limits'; use --help for help")
		}
		return []byte{}, nil
	case "ct-del-limits":
		if cli.Version < 10 {
			return []byte{}, errors.New("ovs-dpctl: unknown command 'ct-get-limits'; use --help for help")
		}
		return []byte{}, nil
	default:
		return []byte{}, nil
	}
}

func TestDPCTLVersion(t *testing.T) {
	var tests = []struct {
		desc    string
		dp      *DataPathService
		version string
	}{
		{
			desc: "Test ovs-dpctl --version ",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: uint8(10),
				},
			},
			version: "ovs-dpctl (Open vSwitch) 2.10.7",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			version, err := tt.dp.Version()
			if err != nil {
				t.Errorf("getting an error while trying to get version %q", err.Error())
			}
			if version != tt.version {
				t.Errorf("want %q but got %q", tt.version, version)
			}
		})
	}
}

func TestGetDataPaths(t *testing.T) {
	var tests = []struct {
		desc    string
		dp      *DataPathService
		dpNames []string
	}{
		{
			desc: "Test ovs-dpctl dump-dps ",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: uint8(10),
				},
			},
			dpNames: []string{"system@ovs-system", "system@dp-test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result, err := tt.dp.GetDataPaths()
			if err != nil {
				t.Errorf("getting an error while trying to get version %q", err.Error())
			}
			for i := range result {
				if result[i] == tt.dpNames[i] {
					t.Log("data path name matches")
				}
			}
		})
	}
}

func TestAddDataPath(t *testing.T) {
	var tests = []struct {
		desc   string
		dp     *DataPathService
		dpName string
	}{
		{
			desc: "Test ovs-dpctl dump-dps ",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: uint8(10),
				},
			},
			dpName: "test2-datapath",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.dp.AddDataPath(tt.dpName)
			if err != nil {
				t.Errorf("getting an error while trying to get version %q", err.Error())
			}
		})
	}
}

func TestDelDataPath(t *testing.T) {
	var tests = []struct {
		desc   string
		dp     *DataPathService
		dpName string
	}{
		{
			desc: "Test ovs-dpctl dump-dps ",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: uint8(10),
				},
			},
			dpName: "test-datapath",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.dp.DelDataPath(tt.dpName)
			if err != nil {
				t.Errorf("getting an error while trying to get version %q", err.Error())
			}
		})
	}
}

func TestGetCTLimits(t *testing.T) {
	d := make(CTLimit)
	d["default"] = 0
	z1 := make(CTLimit)
	z1["zone"] = 2
	z1["count"] = 0
	z1["limit"] = 1000000
	z2 := make(CTLimit)
	z2["zone"] = 3
	z2["count"] = 0
	z2["limit"] = 1000000
	zones := make([]CTLimit, 0)
	zones = append(zones, z1)
	zones = append(zones, z2)
	out := &ConntrackOutput{
		defaultLimit: d,
		zoneLimits:   zones,
	}

	var tests = []struct {
		desc     string
		dp       *DataPathService
		dpName   string
		zones    []uint64
		want     *ConntrackOutput
		err      string
		testCase uint8
	}{
		{
			desc: "Test invalid ovs-dpctl version ",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: unknownCommand,
				},
			},
			dpName:   "ovs-system",
			zones:    []uint64{2, 3},
			want:     nil,
			err:      "ovs-dpctl: unknown command 'ct-get-limits'; use --help for help",
			testCase: unknownCommand,
		},
		{
			desc: "Test valid ovs-dpctl version ",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: 10,
				},
			},
			dpName:   "ovs-system",
			zones:    []uint64{2, 3},
			want:     out,
			err:      "",
			testCase: validTest,
		},
		{
			desc: "Test valid ovs-dpctl ct-get-limits",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: 10,
				},
			},
			dpName:   "",
			zones:    []uint64{},
			want:     out,
			err:      "datapath name argument is mandatory",
			testCase: handleError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			results, err := tt.dp.GetCTLimits(tt.dpName, tt.zones)
			switch tt.testCase {
			case unknownCommand:
				if err.Error() != tt.err {
					t.Errorf("getting an error while trying to get conntrack limits %q", err.Error())
				}
			case validTest:
				if tt.want.defaultLimit["default"] != results.defaultLimit["default"] {
					t.Errorf("mismatched values, want %q  but got %q", tt.want.defaultLimit["default"], results.defaultLimit["default"])
				}
				for i, z := range results.zoneLimits {
					if tt.want.zoneLimits[i]["zone"] != z["zone"] {
						t.Errorf("mismatched values, want %q  but got %q", tt.want.zoneLimits[i]["zone"], z["zone"])
					}
				}
			case handleError:
				if err != nil && err.Error() != tt.err {
					t.Error(err.Error())
				}
			default:
				t.Log("pass")
			}
		})
	}
}

func TestGetCTLimitsWithBinary(t *testing.T) {
	if testing.Short() {
		// if you want to run tests in a context where
		// you don't get ovs-dpctl installed, you should run
		// go test -short to skip it
		t.Skip("skipping test in short mode.")
	}

	d := make(CTLimit)
	d["default"] = 200
	z1 := make(CTLimit)
	z1["zone"] = 2
	z1["count"] = 0
	z1["limit"] = 200
	zones1 := make([]CTLimit, 0)
	zones1 = append(zones1, z1)
	out := &ConntrackOutput{
		defaultLimit: d,
		zoneLimits:   zones1,
	}
	var tests = []struct {
		desc     string
		dp       *DataPathService
		dpName   string
		zones    []uint64
		want     *ConntrackOutput
		err      string
		testCase uint8
	}{
		{
			desc:     "Test valid ovs-dpctl ct-get-limits system@ovs-system zone=2,3 ",
			dp:       NewDataPathService(),
			dpName:   "ovs-system",
			zones:    []uint64{2},
			want:     out,
			err:      "",
			testCase: validTest,
		},
		{
			desc:     "Test valid ovs-dpctl ct-get-limits system@ovs-system",
			dp:       NewDataPathService(),
			dpName:   "ovs-system",
			zones:    []uint64{},
			want:     out,
			err:      "",
			testCase: validTest,
		},
		{
			desc:     "Test valid ovs-dpctl ct-get-limits",
			dp:       NewDataPathService(),
			dpName:   "",
			zones:    []uint64{},
			want:     out,
			err:      "datapath name argument is mandatory",
			testCase: handleError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			results, err := tt.dp.GetCTLimits(tt.dpName, tt.zones)
			switch tt.testCase {
			case validTest:
				if tt.want.defaultLimit["default"] != results.defaultLimit["default"] {
					t.Errorf("mismatched values, want %q  but got %q", tt.want.defaultLimit["default"], results.defaultLimit["default"])
				}
				for i, z := range results.zoneLimits {
					if tt.want.zoneLimits[i]["zone"] != z["zone"] {
						t.Errorf("mismatched values, want %q  but got %q", tt.want.zoneLimits[i]["zone"], z["zone"])
					}
				}
			case handleError:
				if err != nil && err.Error() != tt.err {
					t.Error(err.Error())
				}

			default:
				t.Log("pass")
			}
		})
	}
}

func TestCtSetLimitsArgsToString(t *testing.T) {
	defaultZone := make(map[string]uint64)
	defaultZone["default"] = 100000
	specificZone := make(map[string]uint64)
	specificZone["zone"] = 2
	specificZone["limit"] = 22222
	missingZone := make(map[string]uint64)
	missingZone["limit"] = 22222
	missingLimit := make(map[string]uint64)
	missingLimit["zone"] = 2
	invalidZone := make(map[string]uint64)
	invalidZone["zone"] = 2
	invalidZone["limit"] = 22222
	invalidZone["default"] = 1000

	var tests = []struct {
		desc  string
		zone  map[string]uint64
		want1 string
		want2 string
		err   string
	}{
		{
			desc:  "Test parse valid default argument",
			zone:  defaultZone,
			want1: "default=100000",
			want2: "default=100000",
			err:   "",
		},
		{
			desc:  "Test parse valid zone argument",
			zone:  specificZone,
			want1: "zone=2,limit=22222",
			want2: "limit=22222,zone=2",
			err:   "",
		},
		{
			desc:  "Test parse invalid zone argument",
			zone:  invalidZone,
			want1: "",
			want2: "",
			err:   "missing or too many arguments to setup ct limits",
		},
		{
			desc:  "Test parse missing limit argument",
			zone:  missingLimit,
			want1: "",
			want2: "",
			err:   "wrong argument while setting zone ct limits",
		},
		{
			desc:  "Test parse missing zone argument",
			zone:  missingZone,
			want1: "",
			want2: "",
			err:   "wrong argument while setting zone ct limits",
		},
		{
			desc:  "Test empty zone argument",
			zone:  make(map[string]uint64),
			want1: "",
			want2: "",
			err:   "missing or too many arguments to setup ct limits",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ctSetLimitsArgsToString(tt.zone)
			if err != nil {
				if err.Error() != tt.err {
					t.Error(err.Error())
				}
			}
			if got != tt.want1 && got != tt.want2 {
				t.Errorf("error while getting valid arguments want %q or %q but got %q", tt.want2, tt.want1, got)
			}
		})
	}
}

func TestSetCTLimits(t *testing.T) {
	defaultZone := make(map[string]uint64)
	defaultZone["default"] = 200
	specificZone := make(map[string]uint64)
	specificZone["zone"] = 4
	specificZone["limit"] = 4000

	var tests = []struct {
		desc   string
		dp     *DataPathService
		zone   map[string]uint64
		dpName string
		want   string
		err    string
	}{
		{
			desc: "Test parse valid default argument",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: 10,
				},
			},
			zone:   defaultZone,
			dpName: "system@ovs-system",
		},
		{
			desc: "Test parse valid default argument",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: 10,
				},
			},
			zone:   specificZone,
			dpName: "system@ovs-system",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := tt.dp.SetCTLimits(tt.dpName, tt.zone)
			if err != nil {
				t.Errorf("got %q and %q", res, err.Error())
			}
		})
	}
}

func TestSetCTLimitsWithBinary(t *testing.T) {
	if testing.Short() {
		// if you want to run tests in a context where
		// you don't get ovs-dpctl installed, you should run
		// go test -short to skip it
		t.Skip("skipping test in short mode.")
	}
	defaultZone := make(map[string]uint64)
	defaultZone["default"] = 200
	specificZone := make(map[string]uint64)
	specificZone["zone"] = 4
	specificZone["limit"] = 4000

	var tests = []struct {
		desc   string
		dp     *DataPathService
		zone   map[string]uint64
		dpName string
		want   string
		err    string
	}{
		{
			desc:   "Test parse valid default argument",
			dp:     NewDataPathService(),
			zone:   defaultZone,
			dpName: "system@ovs-system",
		},
		{
			desc:   "Test parse valid default argument",
			dp:     NewDataPathService(),
			zone:   specificZone,
			dpName: "system@ovs-system",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := tt.dp.SetCTLimits(tt.dpName, tt.zone)
			if err != nil {
				t.Errorf("got %q and %q", res, err.Error())
			}
		})
	}
}

func TestDelCTLimits(t *testing.T) {
	var tests = []struct {
		desc     string
		dp       *DataPathService
		zones    []uint64
		dpName   string
		err      string
		testCase uint8
	}{
		{
			desc: "Test del limit with valid argument",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: 10,
				},
			},
			zones:    []uint64{4, 3},
			dpName:   "system@ovs-system",
			testCase: validTest,
		},
		{
			desc: "Test del limit with datapath missing",
			dp: &DataPathService{
				CLI: &MockOvsCLI{
					Version: 10,
				},
			},
			zones:    []uint64{4, 3},
			dpName:   "",
			err:      "datapath name argument is mandatory",
			testCase: handleError,
		},
		{
			desc:     "Test del limit with empty paramaters",
			dp:       NewDataPathService(),
			zones:    []uint64{},
			dpName:   "",
			err:      "datapath name argument is mandatory",
			testCase: handleError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := tt.dp.DelCTLimits(tt.dpName, tt.zones)
			switch tt.testCase {
			case validTest:
				if err != nil {
					t.Errorf("got %q and %q", res, err.Error())
				}

			case handleError:
				if err.Error() != tt.err {
					t.Errorf("got %q and %q", res, err.Error())
				}
			}
		})
	}
}

func TestDelCTLimitsWithBinary(t *testing.T) {
	if testing.Short() {
		// if you want to run tests in a context where
		// you don't get ovs-dpctl installed, you should run
		// go test -short to skip it
		t.Skip("skipping test in short mode.")
	}

	var tests = []struct {
		desc     string
		dp       *DataPathService
		zones    []uint64
		dpName   string
		err      string
		testCase uint8
	}{
		{
			desc:     "Test del limit with valid argument",
			dp:       NewDataPathService(),
			zones:    []uint64{4, 3},
			dpName:   "system@ovs-system",
			testCase: validTest,
		},
		{
			desc:     "Test del limit with datapath missing",
			dp:       NewDataPathService(),
			zones:    []uint64{4, 3},
			dpName:   "",
			err:      "datapath name argument is mandatory",
			testCase: handleError,
		},
		{
			desc:     "Test del limit with empty paramaters",
			dp:       NewDataPathService(),
			zones:    []uint64{},
			dpName:   "",
			err:      "datapath name argument is mandatory",
			testCase: handleError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := tt.dp.DelCTLimits(tt.dpName, tt.zones)
			switch tt.testCase {
			case validTest:
				if err != nil {
					t.Errorf("got %q and %q", res, err.Error())
				}

			case handleError:
				if err.Error() != tt.err {
					t.Errorf("got %q and %q", res, err.Error())
				}
			}
		})
	}
}
