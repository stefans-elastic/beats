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

package idxmgmt

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/lifecycle"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/template"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// SupportFactory is used to provide custom index management support to libbeat.
type SupportFactory func(*logp.Logger, beat.Info, *config.C) (Supporter, error)

// Supporter provides index management and configuration related services
// throughout libbeat.
// The BuildSelector is used by the output to create an IndexSelector. The
// index selector will report the per event index name to be used.
// A manager instantiated via Supporter is responsible for instantiating/configuring
// the index throughout the Elastic Stack.
type Supporter interface {
	// Enabled checks if index management is configured to setup templates or ILM
	Enabled() bool

	// BuildSelector create an index selector.
	// The defaultIndex string is interpreted as format string. It is used
	// as default index if the configuration provided does not define an index or
	// has no default fallback if all indices are guarded by conditionals.
	BuildSelector(cfg *config.C) (outputs.IndexSelector, error)

	// Manager creates a new manager that can be used to execute the required steps
	// for initializing an index, ILM policies, and write aliases.
	Manager(client ClientHandler, assets Asseter) Manager
}

// Asseter provides access to beats assets required to load the template.
type Asseter interface {
	Fields(name string) []byte
}

// Manager is used to initialize indices, ILM policies, and aliases within the
// Elastic Stack.
type Manager interface {
	VerifySetup(template, ilm LoadMode) (bool, string)
	// When supporting index lifecycle management, ensure templates and policies
	// are created before write aliases, to ensure templates are applied to the indices.
	Setup(template, ilm LoadMode) error
}

// LoadMode defines the mode to be used for loading idxmgmt related information.
// It will be used in combination with idxmgmt configuration settings.
type LoadMode uint8

//go:generate stringer -linecomment -type LoadMode
const (
	// LoadModeUnset indicates that no specific mode is set.
	// Instead the decision about loading data will be derived from the config or their respective default values.
	LoadModeUnset LoadMode = iota //unset
	// LoadModeDisabled indicates no loading
	LoadModeDisabled //disabled
	// LoadModeEnabled indicates loading if not already available
	LoadModeEnabled //enabled
	// LoadModeOverwrite indicates overwriting existing components, if loading is not generally disabled.
	LoadModeOverwrite //overwrite
	// LoadModeForce indicates forcing to load components in any case, independent of general loading configurations.
	LoadModeForce //force
)

// Enabled returns whether or not the LoadMode should be considered enabled
func (m *LoadMode) Enabled() bool {
	return m == nil || *m != LoadModeDisabled
}

// DefaultSupport initializes the default index management support used by most Beats.
func DefaultSupport(info beat.Info, configRoot *config.C) (Supporter, error) {
	factory := MakeDefaultSupport(nil, info.Logger)
	return factory(info.Logger, info, configRoot)
}

// MakeDefaultSupport creates some default index management support, with a
// custom ILM support implementation.
func MakeDefaultSupport(ilmSupport lifecycle.SupportFactory, logger *logp.Logger) SupportFactory {
	if ilmSupport == nil {
		ilmSupport = lifecycle.DefaultSupport
	}

	return func(log *logp.Logger, info beat.Info, configRoot *config.C) (Supporter, error) {
		const logName = "index-management"
		log = log.Named(logName)

		// now that we have the "correct" default, unpack the rest of the config
		cfg := struct {
			Lifecycle lifecycle.RawConfig `config:",inline"`
			Template  *config.C           `config:"setup.template"`
			Output    config.Namespace    `config:"output"`
			Migration *config.C           `config:"migration.6_to_7"`
		}{}
		if configRoot != nil {
			if err := configRoot.Unpack(&cfg); err != nil {
				return nil, fmt.Errorf("error unpacking cfg settings while setting up index support: %w", err)
			}
		}

		// consider lifecycles enabled if the user has explicitly enabled them,
		// or if no `enabled` setting has been set by the user, thus reverting to a default of enabled.
		enabled := cfg.Lifecycle.DSL.Enabled() || cfg.Lifecycle.ILM.Enabled() ||
			((cfg.Lifecycle.DSL == nil || !cfg.Lifecycle.DSL.HasField("enabled")) &&
				(cfg.Lifecycle.ILM == nil || !cfg.Lifecycle.ILM.HasField("enabled")))

		if err := checkTemplateESSettings(cfg.Template, cfg.Output); err != nil {
			return nil, err
		}

		return newIndexSupport(log, info, ilmSupport, cfg.Template, enabled, cfg.Migration.Enabled())
	}
}

// checkTemplateESSettings validates template settings and output.elasticsearch
// settings to be consistent.
// XXX: This is some legacy check that will not be active if the output is
// configured via Central Config Management.
// In the future we will have CM deal with index setup and providing a
// consistent output configuration.
//
// TODO: check if it's safe to move this check to the elasticsearch output
// (Not doing so, so to not interfere with outputs being setup via Central
// Management for now).
func checkTemplateESSettings(tmpl *config.C, out config.Namespace) error {
	if out.Name() != "elasticsearch" {
		return nil
	}

	enabled := tmpl == nil || tmpl.Enabled()
	if !enabled {
		return nil
	}

	var tmplCfg template.TemplateConfig
	if tmpl != nil {
		if err := tmpl.Unpack(&tmplCfg); err != nil {
			return fmt.Errorf("unpacking template config fails: %w", err)
		}
	}

	esCfg := struct {
		Index string `config:"index"`
	}{}
	if err := out.Config().Unpack(&esCfg); err != nil {
		return err
	}

	tmplSet := tmplCfg.Name != "" && tmplCfg.Pattern != ""
	if esCfg.Index != "" && !tmplSet {
		return errors.New("setup.template.name and setup.template.pattern have to be set if index name is modified")
	}

	return nil
}
