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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fleetmode"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Evil package level globals
var (
	once        sync.Once
	configfiles *config.StringsFlag
	overwrites  *config.C
	defaults    *config.C
	homePath    *string
	configPath  *string
)

func Initialize() {
	once.Do(func() {
		// The default config cannot include the beat name as
		// it is not initialized when this variable is
		// created. See ChangeDefaultCfgfileFlag which should
		// be called prior to flags.Parse().
		configfiles = config.StringArrFlag(nil, "c", "beat.yml", "Configuration file, relative to path.config")
		overwrites = config.SettingFlag(nil, "E", "Configuration overwrite")
		defaults = config.MustNewConfigFrom(map[string]interface{}{
			"path": map[string]interface{}{
				"home":   ".", // to be initialized by beat
				"config": "${path.home}",
				"data":   filepath.Join("${path.home}", "data"),
				"logs":   filepath.Join("${path.home}", "logs"),
			},
		})
		homePath = config.ConfigOverwriteFlag(nil, overwrites, "path.home", "path.home", "", "Home path")
		configPath = config.ConfigOverwriteFlag(nil, overwrites, "path.config", "path.config", "", "Configuration path")
		_ = config.ConfigOverwriteFlag(nil, overwrites, "path.data", "path.data", "", "Data path")
		_ = config.ConfigOverwriteFlag(nil, overwrites, "path.logs", "path.logs", "", "Logs path")
	})
}

// OverrideChecker checks if a config should be overwritten.
type OverrideChecker func(*config.C) bool

// ConditionalOverride stores a config which needs to overwrite the existing config based on the
// result of the Check.
type ConditionalOverride struct {
	Check  OverrideChecker
	Config *config.C
}

// ChangeDefaultCfgfileFlag replaces the value and default value for
// the `-c` flag so that it reflects the beat name.  It will call
// Initialize() to register the `-c` flags
func ChangeDefaultCfgfileFlag(beatName string) error {
	Initialize()
	configfiles.SetDefault(beatName + ".yml")
	return nil
}

// GetDefaultCfgfile gets the full path of the default config file. Understood
// as the first value for the `-c` flag. By default this will be `<beatname>.yml`
func GetDefaultCfgfile() string {
	if len(configfiles.List()) == 0 {
		return ""
	}

	cfg := configfiles.List()[0]
	cfgpath := GetPathConfig()

	if !filepath.IsAbs(cfg) {
		return filepath.Join(cfgpath, cfg)
	}
	return cfg
}

// HandleFlags adapts default config settings based on command line
// flags.  This also stores if -E management.enabled=true was set on
// command line to determine if running the Beat under agent.  It will
// call Initialize() to register the flags like `-E`.
func HandleFlags() error {
	Initialize()
	// default for the home path is the binary location
	home, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		if *homePath == "" {
			return fmt.Errorf("The absolute path to %s could not be obtained. %w",
				os.Args[0], err)
		}
		home = *homePath
	}

	_ = defaults.SetString("path.home", -1, home)

	if len(overwrites.GetFields()) > 0 {
		common.PrintConfigDebugf(overwrites, "CLI setting overwrites (-E flag):")
	}

	// Enable check to see if beat is running under Agent
	// This is stored in a package so the modules which don't have
	// access to the config can check this value.
	type management struct {
		Enabled bool `config:"management.enabled"`
	}
	var managementSettings management
	cfgFlag := flag.Lookup("E")
	if cfgFlag == nil {
		fleetmode.SetAgentMode(false)
		return nil
	}
	cfgObject, _ := cfgFlag.Value.(*config.SettingsFlag)
	cliCfg := cfgObject.Config()

	err = cliCfg.Unpack(&managementSettings)
	if err != nil {
		fleetmode.SetAgentMode(false)
		return nil //nolint:nilerr // unpacking failing isn't an error for this case
	}
	fleetmode.SetAgentMode(managementSettings.Enabled)
	return nil
}

// Deprecated: Please use Load().
//
// Read reads the configuration from a YAML file into the given interface
// structure. If path is empty this method reads from the configuration
// file specified by the '-c' command line flag.
func Read(out interface{}, path string) error {
	config, err := Load(path, nil)
	if err != nil {
		return err
	}

	return config.Unpack(out)
}

// Load reads the configuration from a YAML file structure. If path is empty
// this method reads from the configuration file specified by the '-c' command
// line flag.
// This function cares about the underlying fleet setting, and if beats is running with
// the management.enabled flag, Load() will bypass reading a config file, and merely merge any overrides.
func Load(path string, beatOverrides []ConditionalOverride) (*config.C, error) {
	var c *config.C
	var err error

	cfgpath := GetPathConfig()

	if !fleetmode.Enabled() {
		if path == "" {
			list := []string{}
			for _, cfg := range configfiles.List() {
				if !filepath.IsAbs(cfg) {
					list = append(list, filepath.Join(cfgpath, cfg))
				} else {
					list = append(list, cfg)
				}
			}
			c, err = common.LoadFiles(list...)
		} else {
			if !filepath.IsAbs(path) {
				path = filepath.Join(cfgpath, path)
			}
			c, err = common.LoadFile(path)
		}
		if err != nil {
			return nil, err
		}
	} else {
		c = config.NewConfig()
	}

	if beatOverrides != nil {
		merged := defaults
		for _, o := range beatOverrides {
			if o.Check(c) {
				merged, err = config.MergeConfigs(merged, o.Config)
				if err != nil {
					return nil, err
				}
			}
		}
		c, err = config.MergeConfigs(
			merged,
			c,
			overwrites,
		)
		if err != nil {
			return nil, err
		}
	} else {
		c, err = config.MergeConfigs(
			defaults,
			c,
			overwrites,
		)
		if err != nil {
			return nil, err
		}
	}

	common.PrintConfigDebugf(c, "Complete configuration loaded:")
	return c, nil
}

// LoadList loads a list of configs data from the given file.
func LoadList(file string, logger *logp.Logger) ([]*config.C, error) {
	logger.Named("cfgfile").Debugf("Load config from file: %s", file)
	rawConfig, err := common.LoadFile(file)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	var c []*config.C
	err = rawConfig.Unpack(&c)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration from file %s: %w", file, err)
	}

	return c, nil
}

func SetConfigPath(path string) {
	*configPath = path
}

// GetPathConfig returns ${path.config}. If ${path.config} is not set,
// ${path.home} is returned.  It will call Initialize to ensure that
// `path.config` and `path.home` are set.
func GetPathConfig() string {
	Initialize()
	if *configPath != "" {
		return *configPath
	} else if *homePath != "" {
		return *homePath
	}
	// TODO: Do we need this or should we always return *homePath?
	return ""
}
