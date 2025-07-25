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

package stress

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type testOutput struct {
	config     testOutputConfig
	observer   outputs.Observer
	batchCount int
}

type testOutputConfig struct {
	Worker      int            `config:"worker" validate:"min=1"`
	BulkMaxSize int            `config:"bulk_max_size"`
	Retry       int            `config:"retry"`
	MinWait     time.Duration  `config:"min_wait"`
	MaxWait     time.Duration  `config:"max_wait"`
	Queue       conf.Namespace `config:"queue"`
	Fail        struct {
		EveryBatch int
	}
}

var defaultTestOutputConfig = testOutputConfig{
	Worker:      1,
	BulkMaxSize: 64,
}

func init() {
	outputs.RegisterType("test", makeTestOutput)
}

func makeTestOutput(_ outputs.IndexManager, beat beat.Info, observer outputs.Observer, cfg *conf.C) (outputs.Group, error) {
	config := defaultTestOutputConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	clients := make([]outputs.Client, config.Worker)
	for i := range clients {
		client := &testOutput{config: config, observer: observer}
		clients[i] = client
	}

	return outputs.Success(config.Queue, config.BulkMaxSize, config.Retry, nil, beat.Logger, clients...)
}

func (*testOutput) Close() error { return nil }

func (t *testOutput) Publish(_ context.Context, batch publisher.Batch) error {
	config := &t.config

	n := len(batch.Events())
	t.observer.NewBatch(n)

	min := int64(config.MinWait)
	max := int64(config.MaxWait)
	if max > 0 && min < max {
		waitFor := rand.Int64N(max-min) + min

		// TODO: make wait interruptable via `Close`
		time.Sleep(time.Duration(waitFor))
	}

	// fail complete batch
	if config.Fail.EveryBatch > 0 {
		t.batchCount++

		if config.Fail.EveryBatch == t.batchCount {
			t.batchCount = 0
			t.observer.RetryableErrors(n)
			batch.Retry()
			return nil
		}

	}

	// TODO: add support to fail single events at end of batch or randomly

	// ack complete batch
	batch.ACK()
	t.observer.AckedEvents(n)

	return nil
}

func (t *testOutput) String() string {
	return "test"
}
