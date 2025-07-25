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

package conditions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCreateNilCondition(t *testing.T) {
	cond, err := NewCondition(nil, logptest.NewTestingLogger(t, ""))
	assert.Nil(t, cond)
	assert.Error(t, err)
}

func GetCondition(t *testing.T, config Config) Condition {
	cond, err := NewCondition(&config, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)
	return cond
}

func GetConditions(t *testing.T, configs []Config) []Condition {
	conds := []Condition{}

	for _, config := range configs {
		conds = append(conds, GetCondition(t, config))
	}
	assert.True(t, len(conds) == len(configs))

	return conds
}

var secdTestEvent = &beat.Event{
	Timestamp: time.Now(),
	Fields: mapstr.M{
		"proc": mapstr.M{
			"cmdline": "/usr/libexec/secd",
			"cpu": mapstr.M{
				"start_time": "Apr10",
				"system":     1988,
				"total":      6029,
				"total_p":    0.08,
				"user":       4041,
			},
			"name":     "secd",
			"pid":      305,
			"ppid":     1,
			"state":    "running",
			"username": "monica",
			"keywords": []interface{}{"foo", "bar"},
		},
		"tags":  []string{"auditbeat", "prod", "security"},
		"type":  "process",
		"final": false,
	},
}

var httpResponseTestEvent = &beat.Event{
	Timestamp: time.Now(),
	Fields: mapstr.M{
		"@timestamp":    "2015-06-11T09:51:23.642Z",
		"bytes_in":      126,
		"bytes_out":     28033,
		"client_ip":     "127.0.0.1",
		"client_port":   42840,
		"client_proc":   "",
		"client_server": "mar.local",
		"http": mapstr.M{
			"code":           200,
			"content_length": 76985,
			"phrase":         "OK",
		},
		"ip":           "127.0.0.1",
		"method":       "GET",
		"params":       "",
		"path":         "/jszip.min.js",
		"port":         8000,
		"proc":         "",
		"query":        "GET /jszip.min.js",
		"responsetime": 30,
		"server":       "mar.local",
		"status":       "OK",
		"type":         "http",
	},
}

var httpResponseEventIPList = &beat.Event{
	Timestamp: time.Now(),
	Fields: mapstr.M{
		"@timestamp": "2024-12-05T09:51:23.642Z",
		"ecs": mapstr.M{
			"version": "8.11.0",
		},
		"host": mapstr.M{
			"hostname": "testhost",
			"os": mapstr.M{
				"type":     "linux",
				"family":   "debian",
				"version":  "11 (bullseye)",
				"platform": "debian",
			},
			"ip": []string{
				"10.1.0.55",
				"fe80::4001:aff:fe9a:55",
			},
		},
	},
}

func testConfig(t *testing.T, expected bool, event *beat.Event, config *Config) {
	t.Helper()
	cond, err := NewCondition(config, logptest.NewTestingLogger(t, ""))
	if assert.NoError(t, err) {
		assert.Equal(t, expected, cond.Check(event))
	}
}

func TestCombinedCondition(t *testing.T) {
	config := Config{
		OR: []Config{
			{
				Range: &Fields{fields: map[string]interface{}{
					"http.code.gte": 100,
					"http.code.lt":  300,
				}},
			},
			{
				AND: []Config{
					{
						Equals: &Fields{fields: map[string]interface{}{
							"status": 200,
						}},
					},
					{
						Equals: &Fields{fields: map[string]interface{}{
							"type": "http",
						}},
					},
				},
			},
		},
	}

	cond := GetCondition(t, config)

	assert.True(t, cond.Check(httpResponseTestEvent))
}
