// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var CmdTestCfg = `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  hosts:
    - %s
  username: admin
  password: testing
`

func TestCmdTest(t *testing.T) {
	esURL := GetESURL(t, "http")
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(CmdTestCfg, esURL.String()))
	mockbeat.Start("test", "config")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("Config OK", 10*time.Second)
}

func TestCmdTestNoConfig(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.Start("test", "config")
	err := mockbeat.Cmd.Wait()
	require.Error(t, err, "mockbeat must exit with an error")
	require.Equal(t, 1, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
}

func TestCmdTestOutput(t *testing.T) {
	esURL := GetESURL(t, "http")
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(CmdTestCfg, esURL.String()))
	mockbeat.Start("test", "output")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("parse url... OK", 10*time.Second)
	mockbeat.WaitStdOutContains("TLS... WARN secure connection disabled", 10*time.Second)
	mockbeat.WaitStdOutContains("talk to server... OK", 10*time.Second)
}

func TestCmdTestOutputBadHost(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(CmdTestCfg, "badhost:9200"))
	mockbeat.Start("test", "output")
	err := mockbeat.Cmd.Wait()
	require.Error(t, err, "mockbeat must exit with error")
	require.Equal(t, 1, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("parse url... OK", 10*time.Second)
	mockbeat.WaitStdOutContains("dns lookup... ERROR", 10*time.Second)
}
