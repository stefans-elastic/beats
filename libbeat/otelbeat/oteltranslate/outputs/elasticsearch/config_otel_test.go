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

package elasticsearchtranslate

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestToOtelConfig(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("basic config translation", func(t *testing.T) {
		beatCfg := `
hosts:
  - localhost:9200
  - localhost:9300
protocol: http
path: /foo/bar
username: elastic
password: changeme
index: "some-index"
pipeline: "some-ingest-pipeline"
proxy_url: "https://proxy.url"
backoff:
  init: 42s
  max: 420s
workers: 30
headers:
  X-Header-1: foo
  X-Bar-Header: bar`

		OTelCfg := `
endpoints:
  - http://localhost:9200/foo/bar
  - http://localhost:9300/foo/bar
idle_conn_timeout: 3s
logs_index: some-index
password: changeme
pipeline: some-ingest-pipeline
proxy_url: https://proxy.url
retry:
  enabled: true
  initial_interval: 42s
  max_interval: 7m0s
  max_retries: 3
timeout: 1m30s
user: elastic
headers:
  X-Header-1: foo
  X-Bar-Header: bar
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
mapping:
  mode: bodymap
compression: gzip
compression_params:
  level: 1
 `
		input := newFromYamlString(t, beatCfg)
		cfg := config.MustNewConfigFrom(input.ToStringMap())
		got, err := ToOTelConfig(cfg, logger)
		require.NoError(t, err, "error translating elasticsearch output to ES exporter config")
		expOutput := newFromYamlString(t, OTelCfg)
		compareAndAssert(t, expOutput, confmap.NewFromStringMap(got))

	})

	t.Run("test api key is encoded before mapping to es-exporter", func(t *testing.T) {
		beatCfg := `
hosts:
  - localhost:9200
index: "some-index"
api_key: "TiNAGG4BaaMdaH1tRfuU:KnR6yE41RrSowb0kQ0HWoA"
`

		OTelCfg := `
endpoints:
  - http://localhost:9200
idle_conn_timeout: 3s
logs_index: some-index
retry:
  enabled: true
  initial_interval: 1s
  max_interval: 1m0s
  max_retries: 3
timeout: 1m30s
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
mapping:
  mode: bodymap  
api_key: VGlOQUdHNEJhYU1kYUgxdFJmdVU6S25SNnlFNDFSclNvd2Iwa1EwSFdvQQ==
compression: gzip
compression_params:
  level: 1
 `
		input := newFromYamlString(t, beatCfg)
		cfg := config.MustNewConfigFrom(input.ToStringMap())
		got, err := ToOTelConfig(cfg, logger)
		require.NoError(t, err, "error translating elasticsearch output to ES exporter config ")
		expOutput := newFromYamlString(t, OTelCfg)
		compareAndAssert(t, expOutput, confmap.NewFromStringMap(got))

	})

	// when preset is configured, we only test worker, bulk_max_size, idle_connection_timeout here
	// es-exporter sets compression level to 1 by default
	t.Run("check preset config translation", func(t *testing.T) {
		commonBeatCfg := `
hosts:
  - localhost:9200
index: "some-index"
username: elastic
password: changeme
preset: %s
`

		commonOTelCfg := `
endpoints:
  - http://localhost:9200
retry:
  enabled: true
  initial_interval: 1s
  max_interval: 1m0s
  max_retries: 3
logs_index: some-index
password: changeme
user: elastic
timeout: 1m30s
mapping:
  mode: bodymap 
compression: gzip
compression_params:
  level: 1
`

		tests := []struct {
			presetName string
			output     string
		}{
			{
				presetName: "balanced",
				output: commonOTelCfg + `
idle_conn_timeout: 3s
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
 `,
			},
			{
				presetName: "throughput",
				output: commonOTelCfg + `
idle_conn_timeout: 15s
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
 `,
			},
			{
				presetName: "scale",
				output: `
endpoints:
  - http://localhost:9200
retry:
  enabled: true
  initial_interval: 5s
  max_interval: 5m0s
  max_retries: 3
logs_index: some-index
password: changeme
user: elastic
timeout: 1m30s
idle_conn_timeout: 1s
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
mapping:
  mode: bodymap    
compression: gzip
compression_params:
  level: 1
 `,
			},
			{
				presetName: "latency",
				output: commonOTelCfg + `
idle_conn_timeout: 1m0s
batcher:
  enabled: true
  max_size: 50
  min_size: 0
 `,
			},
			{
				presetName: "custom",
				output: commonOTelCfg + `
idle_conn_timeout: 3s
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
 `,
			},
		}

		for _, test := range tests {
			t.Run("config translation w/"+test.presetName, func(t *testing.T) {
				input := newFromYamlString(t, fmt.Sprintf(commonBeatCfg, test.presetName))
				cfg := config.MustNewConfigFrom(input.ToStringMap())
				got, err := ToOTelConfig(cfg, logger)
				require.NoError(t, err, "error translating elasticsearch output to OTel ES exporter type")
				expOutput := newFromYamlString(t, test.output)
				compareAndAssert(t, expOutput, confmap.NewFromStringMap(got))
			})
		}

	})

}

func TestCompressionConfig(t *testing.T) {
	compressionConfig := `
hosts:
  - localhost:9200
  - localhost:9300
protocol: http
path: /foo/bar
username: elastic
password: changeme
index: "some-index"
compression_level: %d`

	otelConfig := `
endpoints:
  - http://localhost:9200/foo/bar
  - http://localhost:9300/foo/bar
idle_conn_timeout: 3s
logs_index: some-index
password: changeme
retry:
  enabled: true
  initial_interval: 1s
  max_interval: 1m0s
  max_retries: 3
timeout: 1m30s
user: elastic
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
mapping:
  mode: bodymap
compression: gzip
compression_params:
  level: %d`

	for level := range 9 {
		t.Run(fmt.Sprintf("compression-level-%d", level), func(t *testing.T) {
			input := newFromYamlString(t, fmt.Sprintf(compressionConfig, level))
			cfg := config.MustNewConfigFrom(input.ToStringMap())
			got, err := ToOTelConfig(cfg, logp.NewNopLogger())
			require.NoError(t, err, "error translating elasticsearch output to ES exporter config")
			expOutput := newFromYamlString(t, fmt.Sprintf(otelConfig, level))
			compareAndAssert(t, expOutput, confmap.NewFromStringMap(got))
		})
	}

	t.Run("invalid-compression-level", func(t *testing.T) {
		input := newFromYamlString(t, fmt.Sprintf(compressionConfig, 10))
		cfg := config.MustNewConfigFrom(input.ToStringMap())
		got, err := ToOTelConfig(cfg, logp.NewNopLogger())
		require.ErrorContains(t, err, "failed unpacking config. requires value <= 9 accessing 'compression_level'")
		require.Nil(t, got)
	})

}

func newFromYamlString(t *testing.T, input string) *confmap.Conf {
	t.Helper()
	var rawConf map[string]any
	err := yaml.Unmarshal([]byte(input), &rawConf)
	require.NoError(t, err)

	return confmap.NewFromStringMap(rawConf)
}

func compareAndAssert(t *testing.T, expectedOutput *confmap.Conf, gotOutput *confmap.Conf) {
	t.Helper()
	// convert it to a common type
	want, err := yaml.Marshal(expectedOutput.ToStringMap())
	require.NoError(t, err)
	got, err := yaml.Marshal(gotOutput.ToStringMap())
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got))
}
