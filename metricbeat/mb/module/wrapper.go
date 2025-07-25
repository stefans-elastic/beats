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

package module

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/testing"
)

const (
	// Expvar metric names.
	successesKey           = "success"
	failuresKey            = "failures"
	eventsKey              = "events"
	consecutiveFailuresKey = "consecutive_failures"

	// Failure threshold config key
	failureThresholdKey = "failure_threshold"
)

var (
	fetchesLock = sync.Mutex{}
	fetches     = map[string]*stats{}
)

// Wrapper contains the Module and the private data associated with
// running the Module and its MetricSets.
//
// Use NewWrapper or NewWrappers to construct new Wrappers.
type Wrapper struct {
	mb.Module
	metricSets []*metricSetWrapper // List of pointers to its associated MetricSets.
	monitoring beat.Monitoring

	// Options
	maxStartDelay  time.Duration
	eventModifiers []mb.EventModifier
	logger         *logp.Logger
}

// metricSetWrapper contains the MetricSet and the private data associated with
// running the MetricSet. It contains a pointer to the parent Module.
type metricSetWrapper struct {
	mb.MetricSet
	module *Wrapper // Parent Module.
	stats  *stats   // stats for this MetricSet.

	periodic         bool // Set to true if this metricset is a periodic fetcher
	failureThreshold uint // threshold of consecutive errors needed to set the stream as degraded
}

// stats bundles common metricset stats.
type stats struct {
	key                 string           // full stats key
	ref                 uint32           // number of modules/metricsets reusing stats instance
	success             *monitoring.Int  // Total success events.
	failures            *monitoring.Int  // Total error events.
	events              *monitoring.Int  // Total events published.
	consecutiveFailures *monitoring.Uint // Consecutive failures fetching this metricset
}

// NewWrapper creates a new module and its associated metricsets based on the given configuration.
func NewWrapper(config *conf.C, r *mb.Register, logger *logp.Logger, monitoring beat.Monitoring, options ...Option) (*Wrapper, error) {
	module, metricSets, err := mb.NewModule(config, r, logger)
	if err != nil {
		return nil, err
	}
	return createWrapper(module, metricSets, monitoring, logger, options...)
}

// NewWrapperForMetricSet creates a wrapper for the selected module and metricset.
func NewWrapperForMetricSet(module mb.Module, metricSet mb.MetricSet, monitoring beat.Monitoring, logger *logp.Logger, options ...Option) (*Wrapper, error) {
	return createWrapper(module, []mb.MetricSet{metricSet}, monitoring, logger, options...)
}

func createWrapper(module mb.Module, metricSets []mb.MetricSet, monitoring beat.Monitoring, logger *logp.Logger, options ...Option) (*Wrapper, error) {
	wrapper := &Wrapper{
		Module:     module,
		metricSets: make([]*metricSetWrapper, len(metricSets)),
		monitoring: monitoring,
		logger:     logger,
	}

	for _, applyOption := range options {
		applyOption(wrapper)
	}

	failureThreshold := uint(1)

	var streamHealthSettings struct {
		FailureThreshold *uint `config:"failure_threshold"`
	}

	err := module.UnpackConfig(&streamHealthSettings)

	if err != nil {
		return nil, fmt.Errorf("unpacking raw config: %w", err)
	}

	if streamHealthSettings.FailureThreshold != nil {
		failureThreshold = *streamHealthSettings.FailureThreshold
	}

	for i, metricSet := range metricSets {
		wrapper.metricSets[i] = &metricSetWrapper{
			MetricSet:        metricSet,
			module:           wrapper,
			stats:            getMetricSetStats(monitoring, wrapper.Name(), metricSet.Name()),
			failureThreshold: failureThreshold,
		}
	}
	return wrapper, nil
}

// Wrapper methods

// Start starts the Module's MetricSet workers which are responsible for
// fetching metrics. The workers will continue to periodically fetch until the
// done channel is closed. When the done channel is closed all MetricSet workers
// will stop and the returned output channel will be closed.
//
// The returned channel is buffered with a length one one. It must drained to
// prevent blocking the operation of the MetricSets.
//
// Start should be called only once in the life of a Wrapper.
func (mw *Wrapper) Start(done <-chan struct{}) <-chan beat.Event {
	mw.logger.Named("module").Debugf("Starting %s", mw)

	out := make(chan beat.Event, 1)

	// Start one worker per MetricSet + host combination.
	var wg sync.WaitGroup
	wg.Add(len(mw.metricSets))
	for _, msw := range mw.metricSets {
		go func(msw *metricSetWrapper) {
			metricsPath := msw.ID()
			registry := mw.monitoring.InputsRegistry()

			defer registry.Remove(metricsPath)
			defer releaseStats(mw.monitoring.StatsRegistry(), msw.stats)
			defer wg.Done()
			defer msw.close()

			registry.Add(metricsPath, msw.Metrics(), monitoring.Full)
			monitoring.NewString(msw.Metrics(), "starttime").Set(common.Time(time.Now()).String())

			msw.module.UpdateStatus(status.Starting, fmt.Sprintf("%s/%s is starting", msw.module.Name(), msw.Name()))
			msw.run(done, out)
		}(msw)
	}

	// Close the output channel when all writers to the channel have stopped.
	go func() {
		wg.Wait()
		close(out)
		mw.logger.Named("module").Debugf("Stopped %s", mw)
	}()

	return out
}

// String returns a string representation of Wrapper.
func (mw *Wrapper) String() string {
	return fmt.Sprintf("Wrapper[name=%s, len(metricSetWrappers)=%d]",
		mw.Name(), len(mw.metricSets))
}

// MetricSets return the list of metricsets of the module
func (mw *Wrapper) MetricSets() []*metricSetWrapper {
	return mw.metricSets
}

// metricSetWrapper methods

func (msw *metricSetWrapper) run(done <-chan struct{}, out chan<- beat.Event) {
	defer msw.Logger().Recover(fmt.Sprintf("recovered from panic while fetching "+
		"'%s/%s' for host '%s'", msw.module.Name(), msw.Name(), msw.Host()))

	// Start each metricset randomly over a period of MaxDelayPeriod.
	if msw.module.maxStartDelay > 0 {
		delay := rand.N(msw.module.maxStartDelay)
		msw.Logger().Named("module").Debugf("%v/%v will start after %v", msw.module.Name(), msw.Name(), delay)
		select {
		case <-done:
			return
		case <-time.After(delay):
		}
	}

	msw.Logger().Named("module").Debugf("Starting %s", msw)
	defer msw.Logger().Named("module").Debugf("Stopped %s", msw)

	// Events and errors are reported through this.
	reporter := &eventReporter{
		msw:  msw,
		out:  out,
		done: done,
	}

	switch ms := msw.MetricSet.(type) {
	case mb.PushMetricSet: //nolint:staticcheck // PushMetricSet is deprecated but not removed
		ms.Run(reporter.V1())
	case mb.PushMetricSetV2:
		ms.Run(reporter.V2())
	case mb.PushMetricSetV2WithContext:
		ms.Run(&channelContext{done}, reporter.V2())
	case mb.ReportingMetricSet, mb.ReportingMetricSetV2, mb.ReportingMetricSetV2Error, mb.ReportingMetricSetV2WithContext: //nolint:staticcheck // ReportingMetricSet is deprecated but not removed
		msw.startPeriodicFetching(&channelContext{done}, reporter)
	default:
		// Earlier startup stages prevent this from happening.
		msw.Logger().Errorf("MetricSet '%s/%s' does not implement an event producing interface",
			msw.Module().Name(), msw.Name())
	}
}

// startPeriodicFetching performs an immediate fetch for the MetricSet then it
// begins a continuous timer scheduled loop to fetch data. To stop the loop the
// done channel should be closed.
func (msw *metricSetWrapper) startPeriodicFetching(ctx context.Context, reporter reporter) {
	// Indicate that it has been started as periodic fetcher
	msw.periodic = true

	// Fetch immediately.
	msw.fetch(ctx, reporter)

	// Start timer for future fetches.
	t := time.NewTicker(msw.Module().Config().Period)
	defer t.Stop()
	for {
		select {
		case <-reporter.V2().Done():
			return
		case <-t.C:
			msw.fetch(ctx, reporter)
		}
	}
}

// fetch invokes the appropriate Fetch method for the MetricSet and publishes
// the result using the publisher client. This method will recover from panics
// and log a stack track if one occurs.
func (msw *metricSetWrapper) fetch(ctx context.Context, reporter reporter) {
	switch fetcher := msw.MetricSet.(type) {
	case mb.ReportingMetricSet: //nolint:staticcheck // ReportingMetricSet is deprecated but not removed
		reporter.StartFetchTimer()
		fetcher.Fetch(reporter.V1())
	case mb.ReportingMetricSetV2:
		reporter.StartFetchTimer()
		fetcher.Fetch(reporter.V2())
	case mb.ReportingMetricSetV2Error:
		reporter.StartFetchTimer()
		err := fetcher.Fetch(reporter.V2())
		msw.handleFetchError(err, reporter.V2())
	case mb.ReportingMetricSetV2WithContext:
		reporter.StartFetchTimer()
		err := fetcher.Fetch(ctx, reporter.V2())
		msw.handleFetchError(err, reporter.V2())
	default:
		panic(fmt.Sprintf("unexpected fetcher type for %v", msw))
	}
}

// close closes the underlying MetricSet if it implements the mb.Closer
// interface.
func (msw *metricSetWrapper) close() error {
	if closer, ok := msw.MetricSet.(mb.Closer); ok {
		return closer.Close()
	}
	return nil
}

// String returns a string representation of metricSetWrapper.
func (msw *metricSetWrapper) String() string {
	return fmt.Sprintf("metricSetWrapper[module=%s, name=%s, host=%s]",
		msw.module.Name(), msw.Name(), msw.Host())
}

func (msw *metricSetWrapper) Test(d testing.Driver) {
	d.Run(msw.Name(), func(d testing.Driver) {
		events := make(chan beat.Event, 1)
		done := receiveOneEvent(d, events, msw.module.maxStartDelay+5*time.Second)
		msw.run(done, events)
	})
}

func (msw *metricSetWrapper) handleFetchError(err error, reporter mb.PushReporterV2) {
	switch {
	case err == nil:
		msw.stats.consecutiveFailures.Set(0)
		msw.module.UpdateStatus(status.Running, "")

	case errors.As(err, &mb.PartialMetricsError{}):
		reporter.Error(err)
		msw.stats.consecutiveFailures.Set(0)
		// mark module as running if metrics are partially available and display the error message
		msw.module.UpdateStatus(status.Running, fmt.Sprintf("Error fetching data for metricset %s.%s: %v", msw.module.Name(), msw.Name(), err))
		msw.Logger().Errorf("Error fetching data for metricset %s.%s: %s", msw.module.Name(), msw.Name(), err)

	default:
		reporter.Error(err)
		msw.stats.consecutiveFailures.Inc()
		if msw.failureThreshold > 0 && msw.stats.consecutiveFailures != nil && uint(msw.stats.consecutiveFailures.Get()) >= msw.failureThreshold {
			// mark it as degraded for any other issue encountered
			msw.module.UpdateStatus(status.Degraded, fmt.Sprintf("Error fetching data for metricset %s.%s: %v", msw.module.Name(), msw.Name(), err))
		}
		msw.Logger().Errorf("Error fetching data for metricset %s.%s: %s", msw.module.Name(), msw.Name(), err)

	}
}

type reporter interface {
	StartFetchTimer()
	V1() mb.PushReporter //nolint:staticcheck // PushReporter is deprecated but not removed
	V2() mb.PushReporterV2
}

// eventReporter implements the Reporter interface which is a callback interface
// used by MetricSet implementations to report an event(s), an error, or an error
// with some additional metadata.
type eventReporter struct {
	msw   *metricSetWrapper
	done  <-chan struct{}
	out   chan<- beat.Event
	start time.Time // Start time of the current fetch (or zero for push sources).
}

// startFetchTimer demarcates the start of a new fetch. The elapsed time of a
// fetch is computed based on the time of this call.
func (r *eventReporter) StartFetchTimer() { r.start = time.Now() }
func (r *eventReporter) V1() mb.PushReporter { //nolint:staticcheck // PushReporter is deprecated but not removed
	return reporterV1{v2: r.V2(), module: r.msw.module.Name()}
}
func (r *eventReporter) V2() mb.PushReporterV2 { return reporterV2{r} }

// channelContext implements context.Context by wrapping a channel
type channelContext struct {
	done <-chan struct{}
}

func (r *channelContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (r *channelContext) Done() <-chan struct{}       { return r.done }
func (r *channelContext) Err() error {
	select {
	case <-r.done:
		return context.Canceled
	default:
		return nil
	}
}
func (r *channelContext) Value(key interface{}) interface{} { return nil }

// reporterV1 wraps V2 to provide a v1 interface.
type reporterV1 struct {
	v2     mb.PushReporterV2
	module string
}

func (r reporterV1) Done() <-chan struct{}     { return r.v2.Done() }
func (r reporterV1) Event(event mapstr.M) bool { return r.ErrorWith(nil, event) }
func (r reporterV1) Error(err error) bool      { return r.ErrorWith(err, nil) }
func (r reporterV1) ErrorWith(err error, meta mapstr.M) bool {
	// Skip nil events without error
	if err == nil && meta == nil {
		return true
	}
	return r.v2.Event(mb.TransformMapStrToEvent(r.module, meta, err))
}

type reporterV2 struct {
	*eventReporter
}

func (r reporterV2) Done() <-chan struct{} { return r.done }
func (r reporterV2) Error(err error) bool  { return r.Event(mb.Event{Error: err}) }
func (r reporterV2) Event(event mb.Event) bool {
	if event.Took == 0 && !r.start.IsZero() {
		// ensure elapsed time is always > 0
		event.Took = max(time.Since(r.start), time.Microsecond)
	}
	if r.msw.periodic {
		event.Period = r.msw.Module().Config().Period
	}

	if event.Timestamp.IsZero() {
		if !r.start.IsZero() {
			event.Timestamp = r.start
		} else {
			event.Timestamp = time.Now().UTC()
		}
	}

	if event.Host == "" {
		event.Host = r.msw.HostData().SanitizedURI
	}

	if event.Error == nil {
		r.msw.stats.success.Add(1)
	} else {
		r.msw.stats.failures.Add(1)
	}

	if event.Namespace == "" {
		event.Namespace = r.msw.Registration().Namespace
	}
	beatEvent := event.BeatEvent(r.msw.module.Name(), r.msw.Name(), r.msw.module.eventModifiers...)
	if !writeEvent(r.done, r.out, beatEvent) {
		return false
	}
	r.msw.stats.events.Add(1)

	return true
}

// other utility functions

func writeEvent(done <-chan struct{}, out chan<- beat.Event, event beat.Event) bool {
	select {
	case <-done:
		return false
	case out <- event:
		return true
	}
}

func getMetricSetStats(mon beat.Monitoring, module, name string) *stats {
	key := fmt.Sprintf("metricbeat.%s.%s", module, name)

	fetchesLock.Lock()
	defer fetchesLock.Unlock()

	if s := fetches[key]; s != nil {
		s.ref++
		return s
	}

	reg := mon.StatsRegistry().NewRegistry(key)
	s := &stats{
		key:                 key,
		ref:                 1,
		success:             monitoring.NewInt(reg, successesKey),
		failures:            monitoring.NewInt(reg, failuresKey),
		events:              monitoring.NewInt(reg, eventsKey),
		consecutiveFailures: monitoring.NewUint(reg, consecutiveFailuresKey),
	}

	fetches[key] = s
	return s
}

func releaseStats(reg *monitoring.Registry, s *stats) {
	fetchesLock.Lock()
	defer fetchesLock.Unlock()

	s.ref--
	if s.ref > 0 {
		return
	}

	delete(fetches, s.key)
	reg.Remove(s.key)
}
