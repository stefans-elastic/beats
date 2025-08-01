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

package cfgfile

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/paths"
)

var (
	// DefaultDynamicConfig provides default behavior for a Runner.
	DefaultDynamicConfig = DynamicConfig{
		Reload: Reload{
			Period:  10 * time.Second,
			Enabled: false,
		},
	}

	// configScans measures how many times the config dir was scanned for
	// changes, configReloads measures how many times there were changes that
	// triggered an actual reload.
	configScans   = monitoring.NewInt(nil, "libbeat.config.scans")
	configReloads = monitoring.NewInt(nil, "libbeat.config.reloads")
	moduleStarts  = monitoring.NewInt(nil, "libbeat.config.module.starts")
	moduleStops   = monitoring.NewInt(nil, "libbeat.config.module.stops")
	moduleRunning = monitoring.NewInt(nil, "libbeat.config.module.running") // Number of modules in the runner list (not necessarily in the running state).
)

// DynamicConfig loads config files from a given path, allowing to reload new changes
// while running the beat
type DynamicConfig struct {
	// If path is a relative path, it is relative to the ${path.config}
	Path   string `config:"path"`
	Reload Reload `config:"reload"`
}

// Reload defines reload behavior and frequency
type Reload struct {
	Period  time.Duration `config:"period"`
	Enabled bool          `config:"enabled"`
}

// RunnerFactory is used for validating generated configurations and creating
// of new Runners
type RunnerFactory interface {
	// Create creates a new Runner based on the given configuration.
	Create(p beat.PipelineConnector, config *config.C) (Runner, error)

	// CheckConfig tests if a configuration can be used to create an input. If it
	// is not possible to create an input using the configuration, an error must
	// be returned.
	CheckConfig(config *config.C) error
}

// Runner is a simple interface providing a simple way to
// Start and Stop Reloader
type Runner interface {
	// We include fmt.Stringer here because we do log debug messages that must print
	// something for the given Runner. We need Runner implementers to consciously implement a
	// String() method because the default behavior of `%s` is to print everything recursively
	// in a struct, which could cause a race that would cause the race detector to fail.
	// This is something that could be anticipated for the Runner interface specifically, because
	// most runners will use a goroutine that modifies internal state.
	fmt.Stringer
	Start()
	Stop()
}

// Reloader is used to register and reload modules
type Reloader struct {
	pipeline beat.PipelineConnector
	config   DynamicConfig
	path     string
	done     chan struct{}
	wg       sync.WaitGroup
	logger   *logp.Logger
}

// NewReloader creates new Reloader instance for the given config
func NewReloader(logger *logp.Logger, pipeline beat.PipelineConnector, cfg *config.C) *Reloader {
	conf := DefaultDynamicConfig
	_ = cfg.Unpack(&conf)

	path := conf.Path
	if !filepath.IsAbs(path) {
		path = paths.Resolve(paths.Config, path)
	}

	return &Reloader{
		pipeline: pipeline,
		config:   conf,
		path:     path,
		done:     make(chan struct{}),
		logger:   logger,
	}
}

// Check configs are valid (only if reload is disabled)
func (rl *Reloader) Check(runnerFactory RunnerFactory) error {
	// If config reload is enabled we ignore errors (as they may be fixed afterwards)
	if rl.config.Reload.Enabled {
		return nil
	}

	rl.logger.Debugf("Checking module configs from: %s", rl.path)
	gw := NewGlobWatcher(rl.path, rl.logger)

	files, _, err := gw.Scan()
	if err != nil {
		return fmt.Errorf("fetching config files: %w", err)
	}

	// Load all config objects
	configs, err := rl.loadConfigs(files)
	if err != nil {
		return fmt.Errorf("loading configs: %w", err)
	}

	rl.logger.Debugf("Number of module configs found: %v", len(configs))

	// Initialize modules
	for _, c := range configs {
		// Only add configs to startList which are enabled
		if !c.Config.Enabled() {
			continue
		}

		if err = runnerFactory.CheckConfig(c.Config); err != nil {
			return err
		}
	}
	return nil
}

// Run runs the reloader
func (rl *Reloader) Run(runnerFactory RunnerFactory) {
	rl.logger.Info("Config reloader started")

	list := NewRunnerList("reload", runnerFactory, rl.pipeline, rl.logger)

	rl.wg.Add(1)
	defer rl.wg.Done()

	// Stop all running modules when method finishes
	defer list.Stop()

	gw := NewGlobWatcher(rl.path, rl.logger)

	// If reloading is disable, config files should be loaded immediately
	if !rl.config.Reload.Enabled {
		rl.config.Reload.Period = 0
	}

	// If forceReload is set, the configuration should be reloaded
	// even if there are no changes. It is set on the first iteration,
	// and whenever an attempted reload fails. It is unset whenever
	// a reload succeeds.
	forceReload := true

	for {
		select {
		case <-rl.done:
			rl.logger.Info("Dynamic config reloader stopped")
			return

		case <-time.After(rl.config.Reload.Period):
			rl.logger.Debug("Scan for new config files")
			configScans.Add(1)

			files, updated, err := gw.Scan()
			if err != nil {
				// In most cases of error, updated == false, so will continue
				// to next iteration below
				rl.logger.Errorf("Error fetching new config files: %v", err)
			}

			// if there are no changes, skip this reload unless forceReload is set.
			if !updated && !forceReload {
				continue
			}
			configReloads.Add(1)

			// Load all config objects
			configs, _ := rl.loadConfigs(files)

			rl.logger.Debugf("Number of module configs found: %v", len(configs))

			err = list.Reload(configs)
			// Force reload on the next iteration if and only if the error
			// can be retried.
			// Errors are already logged by list.Reload, so we don't need to
			// propagate details any further.
			forceReload = common.IsInputReloadable(err)
			if forceReload {
				rl.logger.Debugf("error '%v' can be retried. Will try again in %s", err, rl.config.Reload.Period.String())
			} else {
				rl.logger.Debugf("error '%v' cannot retried. Modify any input file to reload.", err)
			}
		}

		// Path loading is enabled but not reloading. Loads files only once and then stops.
		if !rl.config.Reload.Enabled {
			rl.logger.Info("Loading of config files completed.")
			<-rl.done
			rl.logger.Info("Dynamic config reloader stopped")
			return
		}
	}
}

// Load loads configuration files once.
func (rl *Reloader) Load(runnerFactory RunnerFactory) {
	list := NewRunnerList("load", runnerFactory, rl.pipeline, rl.logger)

	rl.wg.Add(1)
	defer rl.wg.Done()

	// Stop all running modules when method finishes
	defer list.Stop()

	gw := NewGlobWatcher(rl.path, rl.logger)

	rl.logger.Debug("Scan for config files")
	files, _, err := gw.Scan()
	if err != nil {
		rl.logger.Errorf("Error fetching new config files: %v", err)
	}

	// Load all config objects
	configs, _ := rl.loadConfigs(files)

	rl.logger.Debugf("Number of module configs found: %v", len(configs))

	if err := list.Reload(configs); err != nil {
		rl.logger.Errorf("Error loading configuration files: %+v", err)
		return
	}

	rl.logger.Info("Loading of config files completed.")
}

func (rl *Reloader) loadConfigs(files []string) ([]*reload.ConfigWithMeta, error) {
	// Load all config objects
	result := []*reload.ConfigWithMeta{}
	var errs []error
	for _, file := range files {
		configs, err := LoadList(file, rl.logger)
		if err != nil {
			errs = append(errs, err)
			rl.logger.Errorf("Error loading config from file '%s', error %v", file, err)
			continue
		}

		for _, c := range configs {
			result = append(result, &reload.ConfigWithMeta{Config: c})
		}
	}

	return result, errors.Join(errs...)
}

// Stop stops the reloader and waits for all modules to properly stop
func (rl *Reloader) Stop() {
	close(rl.done)
	rl.wg.Wait()
}
