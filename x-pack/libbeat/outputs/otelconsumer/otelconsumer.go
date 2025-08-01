// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelconsumer

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/otelbeat/otelmap"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	// esDocumentIDAttribute is the attribute key used to store the document ID in the log record.
	esDocumentIDAttribute = "elasticsearch.document_id"
)

func init() {
	outputs.RegisterType("otelconsumer", makeOtelConsumer)
}

type otelConsumer struct {
	observer     outputs.Observer
	logsConsumer consumer.Logs
	beatInfo     beat.Info
	log          *logp.Logger
}

func makeOtelConsumer(_ outputs.IndexManager, beat beat.Info, observer outputs.Observer, cfg *config.C) (outputs.Group, error) {
	ocConfig := defaultConfig()
	if err := cfg.Unpack(&ocConfig); err != nil {
		return outputs.Fail(err)
	}

	// Default to runtime.NumCPU() workers
	clients := make([]outputs.Client, 0, runtime.NumCPU())
	for range runtime.NumCPU() {
		clients = append(clients, &otelConsumer{
			observer:     observer,
			logsConsumer: beat.LogConsumer,
			beatInfo:     beat,
			log:          beat.Logger.Named("otelconsumer"),
		})
	}

	return outputs.Success(ocConfig.Queue, -1, 0, nil, beat.Logger, clients...)
}

// Close is a noop for otelconsumer
func (out *otelConsumer) Close() error {
	return nil
}

// Publish converts Beat events to Otel format and sends them to the Otel collector
func (out *otelConsumer) Publish(ctx context.Context, batch publisher.Batch) error {
	switch {
	case out.logsConsumer != nil:
		return out.logsPublish(ctx, batch)
	default:
		panic(fmt.Errorf("an otel consumer must be specified"))
	}
}

func (out *otelConsumer) logsPublish(ctx context.Context, batch publisher.Batch) error {
	st := out.observer
	pLogs := plog.NewLogs()
	resourceLogs := pLogs.ResourceLogs().AppendEmpty()
	sourceLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecords := sourceLogs.LogRecords()

	// Convert the batch of events to Otel plog.Logs. The encoding we
	// choose here is to set all fields in a Map in the Body of the log
	// record. Each log record encodes a single beats event.
	// This way we have full control over the final structure of the log in the
	// destination, as long as the exporter allows it.
	// For example, the elasticsearchexporter has an encoding specifically for this.
	// See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/35444.
	events := batch.Events()
	for _, event := range events {
		logRecord := logRecords.AppendEmpty()

		if id, ok := event.Content.Meta["_id"]; ok {
			// Specify the id as an attribute used by the elasticsearchexporter
			// to set the final document ID in Elasticsearch.
			// When using the bodymap encoding in the exporter all attributes
			// are stripped out of the final Elasticsearch document.
			//
			// See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/36882.
			switch id := id.(type) {
			case string:
				logRecord.Attributes().PutStr(esDocumentIDAttribute, id)
			}
		}

		beatEvent := event.Content.Fields
		if beatEvent == nil {
			beatEvent = mapstr.M{}
		}
		beatEvent["@timestamp"] = event.Content.Timestamp
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(event.Content.Timestamp))

		// Set the timestamp for when the event was first seen by the pipeline.
		observedTimestamp := logRecord.Timestamp()
		if created, err := beatEvent.GetValue("event.created"); err == nil {
			switch created := created.(type) {
			case time.Time:
				observedTimestamp = pcommon.NewTimestampFromTime(created)
			case common.Time:
				observedTimestamp = pcommon.NewTimestampFromTime(time.Time(created))
			default:
				out.log.Warnf("Invalid 'event.created' type (%T); using log timestamp as observed timestamp.", created)
			}
		}
		logRecord.SetObservedTimestamp(observedTimestamp)

		otelmap.ConvertNonPrimitive(beatEvent)

		// if data_stream field is set on beatEvent. Add it to logrecord.Attributes to support dynamic indexing
		if val, _ := beatEvent.GetValue("data_stream"); val != nil {
			// If the below sub fields do not exist, it will return empty string.
			subFields := []string{"dataset", "namespace", "type"}

			for _, subField := range subFields {
				// value, ok := data.Map().Get(subField)
				value, err := beatEvent.GetValue("data_stream." + subField)
				if vStr, ok := value.(string); ok && err == nil {
					// set log record attribute only if value is non empty
					logRecord.Attributes().PutStr("data_stream."+subField, vStr)
				}
			}

		}
		if err := logRecord.Body().SetEmptyMap().FromRaw(map[string]any(beatEvent)); err != nil {
			out.log.Errorf("received an error while converting map to plog.Log, some fields might be missing: %v", err)
		}
	}

	err := out.logsConsumer.ConsumeLogs(ctx, pLogs)
	if err != nil {
		// Permanent errors shouldn't be retried. This tipically means
		// the data cannot be serialized by the exporter that is attached
		// to the pipeline or when the destination refuses the data because
		// it cannot decode it. Retrying in this case is useless.
		//
		// See https://github.com/open-telemetry/opentelemetry-collector/blob/1c47d89/receiver/doc.go#L23-L40
		if consumererror.IsPermanent(err) {
			st.PermanentErrors(len(events))
			batch.Drop()
		} else {
			st.RetryableErrors(len(events))
			batch.Retry()
		}

		return fmt.Errorf("failed to send batch events to otel collector: %w", err)
	}

	batch.ACK()
	st.NewBatch(len(events))
	st.AckedEvents(len(events))
	return nil
}

func (out *otelConsumer) String() string {
	return "otelconsumer"
}
