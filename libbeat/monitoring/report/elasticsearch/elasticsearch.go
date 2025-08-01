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

package elasticsearch

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"strconv"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/monitoring/report"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type reporter struct {
	done       *stopper
	logger     *logp.Logger
	monitoring beat.Monitoring

	checkRetry time.Duration

	// event metadata
	beatMeta mapstr.M
	tags     []string

	// pipeline
	pipeline *pipeline.Pipeline
	client   beat.Client

	out []outputs.NetworkClient
	wg  sync.WaitGroup
}

const logSelector = "monitoring"

var errNoMonitoring = errors.New("xpack monitoring not available")

func init() {
	report.RegisterReporterFactory("elasticsearch", makeReporter)
}

func defaultConfig(settings report.Settings) config {
	c := config{
		Hosts:            nil,
		Protocol:         "http",
		Params:           nil,
		Headers:          nil,
		Username:         "beats_system",
		Password:         "",
		APIKey:           "",
		ProxyURL:         "",
		CompressionLevel: 0,
		MaxRetries:       3,
		MetricsPeriod:    10 * time.Second,
		StatePeriod:      1 * time.Minute,
		BulkMaxSize:      50,
		BufferSize:       50,
		Tags:             nil,
		Backoff: backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		ClusterUUID: settings.ClusterUUID,
		Transport:   httpcommon.DefaultHTTPTransportSettings(),
	}

	if settings.DefaultUsername != "" {
		c.Username = settings.DefaultUsername
	}

	return c
}

func makeReporter(beat beat.Info, mon beat.Monitoring, settings report.Settings, cfg *conf.C) (report.Reporter, error) {
	log := beat.Logger.Named(logSelector)
	config := defaultConfig(settings)
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	// Unset username which is set by default, even if no password is set
	if config.APIKey != "" {
		config.Username = ""
		config.Password = ""
	}

	// check endpoint availability on startup only every 30 seconds
	checkRetry := 30 * time.Second
	windowSize := config.BulkMaxSize - 1
	if windowSize <= 0 {
		windowSize = 1
	}

	params := makeClientParams(config)

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.New("empty hosts list")
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		client, err := makeClient(host, params, &config, beat)
		if err != nil {
			return nil, err
		}
		clients[i] = client
	}

	outClient := outputs.NewFailoverClient(clients)
	outClient = outputs.WithBackoff(outClient, config.Backoff.Init, config.Backoff.Max)

	processing, err := processing.MakeDefaultSupport(true, nil)(beat, log, conf.NewConfig())
	if err != nil {
		return nil, err
	}

	queueConfig := conf.Namespace{}
	conf, err := conf.NewConfigFrom(map[string]interface{}{
		"mem.events":           32,
		"mem.flush.min_events": 1,
	})
	if err != nil {
		return nil, err
	}
	err = queueConfig.Unpack(conf)
	if err != nil {
		return nil, err
	}

	pipeline, err := pipeline.New(
		beat,
		pipeline.Monitors{
			Metrics: mon.StatsRegistry().GetOrCreateRegistry("monitoring"),
			Logger:  log,
		},
		queueConfig,
		outputs.Group{
			Clients:   []outputs.Client{outClient},
			BatchSize: windowSize,
			Retry:     0, // no retry. Drop event on error.
		},
		pipeline.Settings{
			WaitClose:     0,
			WaitCloseMode: pipeline.NoWaitOnClose,
			Processors:    processing,
		})
	if err != nil {
		return nil, err
	}

	pipeConn, err := pipeline.Connect()
	if err != nil {
		pipeline.Close()
		return nil, err
	}

	r := &reporter{
		logger:     log,
		monitoring: mon,
		done:       newStopper(),
		beatMeta:   makeMeta(beat),
		tags:       config.Tags,
		checkRetry: checkRetry,
		pipeline:   pipeline,
		client:     pipeConn,
		out:        clients,
	}
	r.wg.Add(1)
	go r.initLoop(config)
	return r, nil
}

func (r *reporter) Stop() {
	r.done.Stop()
	r.client.Close()
	r.pipeline.Close()
	r.wg.Wait()
}

func (r *reporter) initLoop(c config) {
	r.logger.Debug("Start monitoring endpoint init loop.")
	defer func() {
		r.logger.Debug("Finish monitoring endpoint init loop.")
		r.wg.Done()
	}()

	log := r.logger

	logged := false

	for {
		// Select one configured endpoint by random and check if xpack is available
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		client := r.out[rand.IntN(len(r.out))]
		err := client.Connect(ctx)
		if err == nil {
			closing(log, client)
			break
		} else {
			if !logged {
				log.Info("Failed to connect to Elastic X-Pack Monitoring. Either Elasticsearch X-Pack monitoring is not enabled or Elasticsearch is not available. Will keep retrying. Error: ", err)
				logged = true
			}
			r.logger.Debugf("Monitoring could not connect to Elasticsearch, failed with %+v", err)
		}

		select {
		case <-r.done.C():
			return
		case <-time.After(r.checkRetry):
		}
	}

	log.Info("Successfully connected to X-Pack Monitoring endpoint.")

	// Start collector and send loop if monitoring endpoint has been found.
	go r.snapshotLoop(r.monitoring.StateRegistry(), "state", "state", c.StatePeriod, c.ClusterUUID)
	// For backward compatibility stats is named to metrics.
	go r.snapshotLoop(r.monitoring.StatsRegistry(), "stats", "metrics", c.MetricsPeriod, c.ClusterUUID)
}

func (r *reporter) snapshotLoop(registry *monitoring.Registry, namespace, prefix string, period time.Duration, clusterUUID string) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	log := r.logger

	log.Infof("Start monitoring %s metrics snapshot loop with period %s.", namespace, period)
	defer log.Infof("Stop monitoring %s metrics snapshot loop.", namespace)

	for {
		var ts time.Time

		select {
		case <-r.done.C():
			return
		case ts = <-ticker.C:
		}

		snapshot := makeSnapshot(registry)
		if snapshot == nil {
			log.Debug("Empty snapshot.")
			continue
		}

		fields := mapstr.M{
			"beat": r.beatMeta,
			prefix: snapshot,
		}
		if len(r.tags) > 0 {
			fields["tags"] = r.tags
		}

		meta := mapstr.M{
			"type":        "beats_" + namespace,
			"interval_ms": int64(period / time.Millisecond),
			// Converting to seconds as interval only accepts `s` as unit
			"params": map[string]string{"interval": strconv.Itoa(int(period/time.Second)) + "s"},
		}

		if clusterUUID == "" {
			clusterUUID = getClusterUUID(r.monitoring)
		}
		if clusterUUID != "" {
			_, _ = meta.Put("cluster_uuid", clusterUUID)
		}

		r.client.Publish(beat.Event{
			Timestamp: ts,
			Fields:    fields,
			Meta:      meta,
		})
	}
}

func makeClient(host string, params map[string]string, config *config, beat beat.Info) (outputs.NetworkClient, error) {
	url, err := common.MakeURL(config.Protocol, "", host, 9200)
	if err != nil {
		return nil, err
	}

	esClient, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              url,
		Beatname:         beat.Beat,
		Username:         config.Username,
		Password:         config.Password,
		APIKey:           config.APIKey,
		Parameters:       params,
		Headers:          config.Headers,
		CompressionLevel: config.CompressionLevel,
		Transport:        config.Transport,
		UserAgent:        beat.UserAgent,
	}, beat.Logger)
	if err != nil {
		return nil, err
	}

	return newPublishClient(esClient, params, beat.Logger)
}

func closing(log *logp.Logger, c io.Closer) {
	if err := c.Close(); err != nil {
		log.Warnf("Closed failed with: %v", err)
	}
}

func makeMeta(beat beat.Info) mapstr.M {
	return mapstr.M{
		"type":    beat.Beat,
		"version": beat.Version,
		"name":    beat.Name,
		"host":    beat.Hostname,
		"uuid":    beat.ID,
	}
}

func getClusterUUID(mon beat.Monitoring) string {
	stateRegistry := mon.StateRegistry()
	outputsRegistry := stateRegistry.GetRegistry("outputs")
	if outputsRegistry == nil {
		return ""
	}

	elasticsearchRegistry := outputsRegistry.GetRegistry("elasticsearch")
	if elasticsearchRegistry == nil {
		return ""
	}

	snapshot := monitoring.CollectFlatSnapshot(elasticsearchRegistry, monitoring.Full, false)
	return snapshot.Strings["cluster_uuid"]
}

func makeClientParams(config config) map[string]string {
	params := map[string]string{}

	for k, v := range config.Params {
		params[k] = v
	}

	return params
}
