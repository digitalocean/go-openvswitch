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
	"regexp"
	"strconv"
	"strings"
)

var (
	errMissingMandatoryDataPathName = errors.New("datapath name argument is mandatory")
	errUninitializedClient          = errors.New("client unitialized")
	errMissingMandatoryZone         = errors.New("at least 1 zone is mandatory")
	errWrongArgumentNumber          = errors.New("missing or too many arguments to setup ct limits")
	errWrongDefaultArgument         = errors.New("wrong argument while setting default ct limits")
	errWrongZoneArgument            = errors.New("wrong argument while setting zone ct limits")
)

// CTLimit defines the type used to store a zone as it is returned
// by ovs-dpctl ct-*-limits commands
type CTLimit map[string]uint64

// ConntrackOutput is a type defined to store the output
// of ovs-dpctl ct-*-limits commands. For example it stores
// such a cli output:
// # ovs-dpctl ct-get-limits system@ovs-system zone=2,3
// default limit=0
// zone=2,limit=0,count=0
// zone=3,limit=0,count=0
type ConntrackOutput struct {
	// defaultLimit is used to store the global setting: default
	defaultLimit CTLimit
	// zones stores all remaning zone's settings
	zoneLimits []CTLimit
}

// DataPathReader is the interface defining the read operations
// for the ovs DataPaths
type DataPathReader interface {
	// Version is the method used to get the version of ovs-dpctl
	Version() (string, error)
	// GetDataPath is the method that returns all DataPaths setup
	// for an ovs switch
	GetDataPath() ([]string, error)
}

// DataPathWriter is the interface defining the wrtie operations
// for the ovs DataPaths
type DataPathWriter interface {
	// AddDataPath is the method used to add a datapath to the switch
	AddDataPath(string) error
	// DelDataPath is the method used to remove a datapath from the switch
	DelDataPath(string) error
}

// ConnTrackReader is the interface defining the read operations
// of ovs conntrack
type ConnTrackReader interface {
	// GetCTLimits is the method used to querying conntrack limits for a
	// datapath on a switch
	GetCTLimits(string, []uint64) (ConntrackOutput, error)
}

// ConnTrackWriter is the interface defining the write operations
// of ovs conntrack
type ConnTrackWriter interface {
	// SetCTLimits is the method used to setup a limit for a zone
	// belonging to a datapath of  a switch
	SetCTLimits(string) (string, error)
	// DelCTLimits is the method used to remove a limit to a zone
	// belonging to a datapath of a switch
	DelCTLimits(string, []uint64) (string, error)
}

// CLI is an interface defining a contract for executing a command.
// Implementation of shell cli is done by the Client concrete type
type CLI interface {
	Exec(args ...string) ([]byte, error)
}

// DataPathService defines the concrete type used for DataPath operations
// supported by the ovs-dpctl command
type DataPathService struct {
	// We define here a CLI interface making easier to mock ovs-dpctl command
	// as in github.com/digitalocean/go-openvswitch/ovs/datapath_test.go
	CLI
}

// NewDataPathService is a builder for the DataPathService.
// sudo is defined as a default option.
func NewDataPathService() *DataPathService {
	return &DataPathService{
		CLI: &DpCLI{
			c: New(Sudo()),
		},
	}
}

// Version retruns the ovs-dptcl --version currently installed
func (dp *DataPathService) Version() (string, error) {
	result, err := dp.CLI.Exec("--version")
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// GetDataPaths returns the output of the command 'ovs-dpctl dump-dps'
func (dp *DataPathService) GetDataPaths() ([]string, error) {
	result, err := dp.CLI.Exec("dump-dps")
	if err != nil {
		return nil, err
	}

	return strings.Split(string(result), "\n"), nil
}

// AddDataPath create a Datapath with the command 'ovs-dpctl add-dp <DP>'
// It takes one argument, the required DataPath Name and returns an error
// if it failed
func (dp *DataPathService) AddDataPath(dpName string) error {
	_, err := dp.CLI.Exec("add-dp", dpName)
	return err
}

// DelDataPath create a Datapath with the command 'ovs-dpctl del-dp <DP>'
// It takes one argument, the required DataPath Name and returns an error
// if it failed
func (dp *DataPathService) DelDataPath(dpName string) error {
	_, err := dp.CLI.Exec("del-dp", dpName)

	return err
}

// GetCTLimits returns the conntrack limits  for a given datapath
// equivalent to running: 'sudo ovs-dpctl ct-get-limits <datapath_name> zone=<#1>,<#2>,...'
func (dp *DataPathService) GetCTLimits(dpName string, zones []uint64) (*ConntrackOutput, error) {
	// Start by building the args
	if dpName == "" {
		return nil, errMissingMandatoryDataPathName
	}

	args := []string{"ct-get-limits", dpName}

	zoneParam := getZoneString(zones)
	if zoneParam != "" {
		args = append(args, zoneParam)
	}

	// call the cli
	results, err := dp.CLI.Exec(args...)
	if err != nil {
		return nil, err
	}

	// Process the results
	entries := strings.Split(string(results), "\n")
	ctOut := &ConntrackOutput{}

	r, err := regexp.Compile(`default`)
	if err != nil {
		return nil, err
	}

	// First start extracting the default conntrack limit setup
	// If found the default value is removed from the entries
	for i, entry := range entries {
		if r.MatchString(entry) {
			ctOut.defaultLimit = make(CTLimit)
			limit, err := strconv.Atoi(strings.Split(entry, "=")[1])
			if err != nil {
				return nil, err
			}
			ctOut.defaultLimit["default"] = uint64(limit)
			// As the default has been found let's remove it
			entries = append(entries[:i], entries[i+1:]...)
		}
	}

	// Now process the zones setup
	for _, entry := range entries {
		fields := strings.Split(entry, ",")
		z := make(CTLimit)
		for _, field := range fields {
			buf := strings.Split(field, "=")
			val, _ := strconv.Atoi(buf[1])
			z[buf[0]] = uint64(val)
		}
		ctOut.zoneLimits = append(ctOut.zoneLimits, z)
	}

	return ctOut, nil
}

// SetCTLimits set the limit for a specific zone or globally.
// Only one zone or default can be set up at once as the cli allows.
// Examples of commands it wrapps:
// sudo ovs-dpctl ct-set-limits system@ovs-system zone=331,limit=1000000
// sudo ovs-dpctl ct-set-limits system@ovs-system default=1000000
func (dp *DataPathService) SetCTLimits(dpName string, zone map[string]uint64) (string, error) {
	// Sanitize the input
	if dpName == "" {
		return "", errMissingMandatoryDataPathName
	}
	argsStr, err := ctSetLimitsArgsToString(zone)
	if err != nil {
		return "", err
	}
	// call the cli
	argsCLI := []string{"ct-set-limits", dpName, argsStr}
	results, err := dp.CLI.Exec(argsCLI...)

	return string(results), err
}

// DelCTLimits deletes limits setup for zones. It takes the Datapath name
// and zones to delete the limits.
// sudo ovs-dpctl  ct-del-limits system@ovs-system  zone=40,4
func (dp *DataPathService) DelCTLimits(dpName string, zones []uint64) (string, error) {
	if dpName == "" {
		return "", errMissingMandatoryDataPathName
	}
	if len(zones) < 1 {
		return "", errMissingMandatoryZone
	}

	var firstZone uint64
	firstZone, zones = zones[0], zones[1:]
	zonesStr := "zone=" + strconv.FormatUint(firstZone, 10)
	for _, z := range zones {
		zonesStr += "," + strconv.FormatUint(z, 10)
	}

	// call the cli
	argsCLI := []string{"ct-del-limits", dpName, zonesStr}
	results, err := dp.CLI.Exec(argsCLI...)

	return string(results), err
}

// ctSetLimitsArgsToString helps formating and sanatizing an input
// It takes a  map and output a string like this:
// - "zone=2,limit=10000" or "limit=10000,zone=2"
// - "default=10000"
func ctSetLimitsArgsToString(zone map[string]uint64) (string, error) {
	defaultSetup := false
	args := make([]string, 0)
	for k, v := range zone {
		if k == "default" {
			args = append(args, k+"="+strconv.FormatUint(v, 10))
			defaultSetup = true
		} else if k == "zone" || k == "limit" {
			args = append(args, k+"="+strconv.FormatUint(v, 10))
		}
	}

	// We need at most 2 arguments and at least 1
	if len(args) == 0 || len(args) > 2 {
		return "", errWrongArgumentNumber

	}
	// if we setup the default global setting we only need a single parameter
	// like "default=100000" and nothing else
	if defaultSetup && len(args) != 1 {
		return "", errWrongDefaultArgument
	}
	// if we setup a limit for dedicated zone we need 2 params like
	// "zone=3" and "limit=50000"
	if !defaultSetup && len(args) != 2 {
		return "", errWrongZoneArgument
	}

	var argsStr string
	argsStr, args = args[0], args[1:]
	if len(args) > 0 {
		for _, s := range args {
			argsStr += "," + s
		}
	}
	return argsStr, nil
}

// getZoneString takes the zones as []uint64 to return a formated
// string usable in different ovs-dpctl commands
// Example a slice: var zones = []uint64{2, 3, 4}
// will output: "zone=2,3,4"
func getZoneString(z []uint64) string {
	zonesStr := make([]string, 0)
	for _, zone := range z {
		zonesStr = append(zonesStr, strconv.FormatUint(zone, 10))
	}

	var sb strings.Builder
	var firstZone string
	if len(zonesStr) > 0 {
		sb.WriteString("zone=")
		firstZone, zonesStr = zonesStr[0], zonesStr[1:]
	}
	sb.WriteString(firstZone)

	for _, zone := range zonesStr {
		sb.WriteString(",")
		sb.WriteString(zone)
	}

	return sb.String()
}

// DpCLI implements the CLI interface by invoking the Client exec
// method.
type DpCLI struct {
	// Wrapped client for ovs-dpctl
	c *Client
}

// Exec executes 'ovs-dpctl' + args passed in argument
func (cli *DpCLI) Exec(args ...string) ([]byte, error) {
	if cli.c == nil {
		return nil, errUninitializedClient
	}

	return cli.c.exec("ovs-dpctl", args...)
}
