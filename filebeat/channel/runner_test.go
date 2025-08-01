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

package channel

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/processors"
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestProcessorsForConfig(t *testing.T) {
	testCases := map[string]struct {
		beatInfo       beat.Info
		configStr      string
		clientCfg      beat.ClientConfig
		event          beat.Event
		expectedFields map[string]string
	}{
		"Simple static index": {
			configStr: "index: 'test'",
			expectedFields: map[string]string{
				"@metadata.raw_index": "test",
			},
		},
		"Index with agent info + timestamp": {
			beatInfo:  beat.Info{Beat: "TestBeat", Version: "3.9.27", Logger: logptest.NewTestingLogger(t, "")},
			configStr: "index: 'beat-%{[agent.name]}-%{[agent.version]}-%{+yyyy.MM.dd}'",
			event:     beat.Event{Timestamp: time.Date(1999, time.December, 31, 23, 0, 0, 0, time.UTC)},
			expectedFields: map[string]string{
				"@metadata.raw_index": "beat-TestBeat-3.9.27-1999.12.31",
			},
		},
		"Set index in ClientConfig": {
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw_index": "clientCfgIndex",
			},
		},
		"ClientConfig processor runs after beat input Index": {
			configStr: "index: 'test'",
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw_index": "clientCfgIndex",
			},
		},
		"Set field in input config": {
			configStr: `processors: [add_fields: {fields: {testField: inputConfig}}]`,
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
		"Set field in ClientConfig": {
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(addfields.NewAddFields(mapstr.M{
						"fields": mapstr.M{"testField": "clientConfig"},
					}, false, true)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "clientConfig",
			},
		},
		"Input config processors run after ClientConfig": {
			configStr: `processors: [add_fields: {fields: {testField: inputConfig}}]`,
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(addfields.NewAddFields(mapstr.M{
						"fields": mapstr.M{"testField": "clientConfig"},
					}, false, true)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
	}
	for description, test := range testCases {
		if test.event.Fields == nil {
			test.event.Fields = mapstr.M{}
		}
		config, err := conf.NewConfigFrom(test.configStr)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}

		editor, err := newCommonConfigEditor(test.beatInfo, config)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}

		clientCfg, err := editor(test.clientCfg)
		require.NoError(t, err)

		processors := clientCfg.Processing.Processor
		processedEvent, err := processors.Run(&test.event)
		// We don't check if err != nil, because we are testing the final outcome
		// of running the processors, including when some of them fail.
		if processedEvent == nil {
			t.Errorf("[%s] Unexpected fatal error running processors: %v\n",
				description, err)
		}
		for key, value := range test.expectedFields {
			field, err := processedEvent.GetValue(key)
			if err != nil {
				t.Errorf("[%s] Couldn't get field %s from event: %v", description, key, err)
				continue
			}
			assert.Equal(t, field, value)
			fieldStr, ok := field.(string)
			if !ok {
				// Note that requiring a string here is just to simplify the test setup,
				// not a requirement of the underlying api.
				t.Errorf("[%s] Field [%s] should be a string", description, key)
				continue
			}
			if fieldStr != value {
				t.Errorf("[%s] Event field [%s]: expected [%s], got [%s]", description, key, value, fieldStr)
			}
		}
	}
}

func TestProcessorsForConfigIsFlat(t *testing.T) {
	// This test is regrettable, and exists because of inconsistencies in
	// processor handling between processors.Processors and processing.group
	// (which implements beat.ProcessorList) -- see processorsForConfig for
	// details. The upshot is that, for now, if the input configuration specifies
	// processors, they must be returned as direct children of the resulting
	// processors.Processors (rather than being collected in additional tree
	// structure).
	// This test should be removed once we have a more consistent mechanism for
	// collecting and running processors.
	configStr := `processors:
- add_fields: {fields: {testField: value}}
- add_fields: {fields: {testField2: stuff}}`
	config, err := conf.NewConfigFrom(configStr)
	if err != nil {
		t.Fatal(err)
	}

	editor, err := newCommonConfigEditor(beat.Info{}, config)
	if err != nil {
		t.Fatal(err)
	}

	clientCfg, err := editor(beat.ClientConfig{})
	require.NoError(t, err)

	lst := clientCfg.Processing.Processor
	assert.Equal(t, 2, len(lst.(*processors.Processors).List)) //nolint:errcheck //Safe to ignore in tests
}

// setRawIndex is a bare-bones processor to set the raw_index field to a
// constant string in the event metadata. It is used to test order of operations
// for processorsForConfig.
type setRawIndex struct {
	indexStr string
}

func (p *setRawIndex) Run(event *beat.Event) (*beat.Event, error) {
	if event.Meta == nil {
		event.Meta = mapstr.M{}
	}
	event.Meta[events.FieldMetaRawIndex] = p.indexStr
	return event, nil
}

func (p *setRawIndex) String() string {
	return fmt.Sprintf("set_raw_index=%v", p.indexStr)
}

// makeProcessors wraps one or more bare Processor objects in Processors.
func makeProcessors(procs ...beat.Processor) *processors.Processors {
	logger, _ := logp.NewDevelopmentLogger("")
	procList := processors.NewList(logger)
	procList.List = procs
	return procList
}

func TestRunnerFactoryWithCommonInputSettings(t *testing.T) {

	// we use `add_kubernetes_metadata` and `add_cloud_metadata`
	// for testing because initially the problem we've discovered
	// was visible with these 2 processors.
	configYAML := `
processors:
  - add_kubernetes_metadata: ~
  - add_cloud_metadata: ~
keep_null: true
publisher_pipeline:
  disable_host: true
type: "filestream"
service.type: "module"
pipeline: "test"
index: "%{[fields.log_type]}-%{[agent.version]}-%{+yyyy.MM.dd}"
`
	// illumos: this specific test requires add_kubernetes_metadata for side-effects
	//   in this test which trigger issues for the stubbed version provided for
	//   illumos (see prior comment about the side-effects being the purpose).
	if runtime.GOOS == "illumos" {
		configYAML = strings.ReplaceAll(configYAML, "\n  - add_kubernetes_metadata: ~", "")
	}
	cfg, err := conf.NewConfigWithYAML([]byte(configYAML), configYAML)
	require.NoError(t, err)

	b := beat.Info{Logger: logptest.NewTestingLogger(t, "")} // not important for the test
	rf := &runnerFactoryMock{
		clientCount: 3, // we will create 3 clients from the wrapped pipeline
	}
	pcm := &pipelineConnectorMock{} // creates mock pipeline clients and will get wrapped

	rfwc := RunnerFactoryWithCommonInputSettings(b, rf)

	// create a wrapped runner, our mock runner will
	// create the given amount of clients here using the wrapped pipeline connector.
	_, err = rfwc.Create(pcm, cfg)
	require.NoError(t, err)

	rf.Assert(t)
}
