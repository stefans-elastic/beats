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

package syslog

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestWhenPriorityIsSet(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	e.SetMessage([]byte("hello world"))
	e.SetHostname([]byte("wopr"))
	e.SetPid([]byte("123"))

	m := dummyMetadata()
	event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))

	expected := mapstr.M{
		"log": mapstr.M{
			"source": mapstr.M{
				"address": "127.0.0.1",
			},
		},
		"message":  "hello world",
		"hostname": "wopr",
		"process": mapstr.M{
			"pid": 123,
		},
		"event": mapstr.M{
			"severity": 5,
		},
		"syslog": mapstr.M{
			"facility":       1,
			"severity_label": "Notice",
			"facility_label": "user-level",
			"priority":       13,
		},
	}

	assert.Equal(t, expected, event.Fields)
}

func TestWhenPriorityIsNotSet(t *testing.T) {
	e := newEvent()
	e.SetMessage([]byte("hello world"))
	e.SetHostname([]byte("wopr"))
	e.SetPid([]byte("123"))

	m := dummyMetadata()
	event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))
	expected := mapstr.M{
		"log": mapstr.M{
			"source": mapstr.M{
				"address": "127.0.0.1",
			},
		},
		"message":  "hello world",
		"hostname": "wopr",
		"process": mapstr.M{
			"pid": 123,
		},
		"event":  mapstr.M{},
		"syslog": mapstr.M{},
	}

	assert.Equal(t, expected, event.Fields)
}

func TestPid(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		e.SetPid([]byte("123"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))
		v, err := event.GetValue("process")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, mapstr.M{"pid": 123}, v)
	})

	t.Run("is not set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))

		_, err := event.GetValue("process")
		assert.Equal(t, mapstr.ErrKeyNotFound, err)
	})
}

func TestHostname(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		e.SetHostname([]byte("wopr"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))
		v, err := event.GetValue("hostname")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, "wopr", v)
	})

	t.Run("is not set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))

		_, err := event.GetValue("hostname")
		if !assert.Error(t, err) {
			return
		}
	})
}

func TestProgram(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		e.SetProgram([]byte("sudo"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))
		v, err := event.GetValue("process")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, mapstr.M{"program": "sudo"}, v)
	})

	t.Run("is not set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))

		_, err := event.GetValue("process")
		assert.Equal(t, mapstr.ErrKeyNotFound, err)
	})
}

func TestSequence(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		e.SetProgram([]byte("sudo"))
		e.SetSequence([]byte("123"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))
		v, err := event.GetValue("event.sequence")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, v, 123)
	})

	t.Run("is not set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logptest.NewTestingLogger(t, "syslog"))

		_, err := event.GetValue("event.sequence")
		assert.Error(t, err)
	})
}

func TestParseAndCreateEvent3164(t *testing.T) {
	cases := map[string]struct {
		data     []byte
		expected mapstr.M
	}{
		"valid data": {
			data: []byte("<34>Oct 11 22:14:15 mymachine su[230]: 'su root' failed for lonvick on /dev/pts/8"),
			expected: mapstr.M{
				"event":    mapstr.M{"severity": 2},
				"hostname": "mymachine",
				"log": mapstr.M{
					"source": mapstr.M{
						"address": "127.0.0.1",
					},
				},
				"message": "'su root' failed for lonvick on /dev/pts/8",
				"process": mapstr.M{"pid": 230, "program": "su"},
				"syslog": mapstr.M{
					"facility":       4,
					"facility_label": "security/authorization",
					"priority":       34,
					"severity_label": "Critical",
				},
			},
		},

		"invalid data": {
			data: []byte("invalid"),
			expected: mapstr.M{
				"log": mapstr.M{
					"source": mapstr.M{
						"address": "127.0.0.1",
					},
				},
				"message": "invalid",
			},
		},
	}

	tz := time.Local
	log := logptest.NewTestingLogger(t, "syslog")
	metadata := dummyMetadata()

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			event := parseAndCreateEvent3164(c.data, metadata, tz, log)
			assert.Equal(t, c.expected, event.Fields)
			assert.Equal(t, metadata.Truncated, event.Meta["truncated"])
		})
	}
}

func TestNewInputDone(t *testing.T) {
	config := mapstr.M{
		"protocol.tcp.host": "localhost:9000",
	}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}

func dummyMetadata() inputsource.NetworkMetadata {
	ip := "127.0.0.1"
	parsedIP := net.ParseIP(ip)
	addr := &net.IPAddr{IP: parsedIP, Zone: ""}
	return inputsource.NetworkMetadata{RemoteAddr: addr}
}

func TestParseAndCreateEvent5424(t *testing.T) {
	cases := map[string]struct {
		data     []byte
		expected mapstr.M
	}{
		"valid data": {
			data: []byte(RfcDoc65Example1),
			expected: mapstr.M{
				"event":    mapstr.M{"severity": 2},
				"hostname": "mymachine.example.com",
				"log": mapstr.M{
					"source": mapstr.M{
						"address": "127.0.0.1",
					},
				},
				"process": mapstr.M{
					"name":      "su",
					"entity_id": "-",
				},
				"message": "'su root' failed for lonvick on /dev/pts/8",
				"syslog": mapstr.M{
					"facility":       4,
					"facility_label": "security/authorization",
					"priority":       34,
					"severity_label": "Critical",
					"msgid":          "ID47",
					"version":        1,
				},
			},
		},
		"valid data2": {
			data: []byte(RfcDoc65Example3),
			expected: mapstr.M{
				"event":    mapstr.M{"severity": 5},
				"hostname": "mymachine.example.com",
				"log": mapstr.M{
					"source": mapstr.M{
						"address": "127.0.0.1",
					},
				},
				"process": mapstr.M{
					"name":      "evntslog",
					"entity_id": "-",
				},
				"message": "An application event log entry...",
				"syslog": mapstr.M{
					"facility":       20,
					"facility_label": "local4",
					"priority":       165,
					"severity_label": "Notice",
					"msgid":          "ID47",
					"version":        1,
					"data": EventData{
						"exampleSDID@32473": {
							"eventID":     "1011",
							"eventSource": "Application",
							"iut":         "3",
						},
					},
				},
			},
		},

		"invalid data": {
			data: []byte("<34>Oct 11 22:14:15 mymachine su[230]: 'su root' failed for lonvick on /dev/pts/8"),
			expected: mapstr.M{
				"log": mapstr.M{
					"source": mapstr.M{
						"address": "127.0.0.1",
					},
				},
				"message": "<34>Oct 11 22:14:15 mymachine su[230]: 'su root' failed for lonvick on /dev/pts/8",
			},
		},
	}

	tz := time.Local
	log := logptest.NewTestingLogger(t, "syslog")
	metadata := dummyMetadata()

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			event := parseAndCreateEvent5424(c.data, metadata, tz, log)
			assert.Equal(t, c.expected, event.Fields)
			assert.Equal(t, metadata.Truncated, event.Meta["truncated"])
		})
	}
}
