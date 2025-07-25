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

package jolokia

import (
	"fmt"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	//nolint:errcheck // init function
	autodiscover.Registry.AddProvider("jolokia", AutodiscoverBuilder)
}

// DiscoveryProber implements discovery probes
type DiscoveryProber interface {
	Start()
	Stop()
	Events() <-chan Event
}

// Provider is the Jolokia Discovery autodiscover provider
type Provider struct {
	bus       bus.Bus
	builders  autodiscover.Builders
	appenders autodiscover.Appenders
	templates template.Mapper
	discovery DiscoveryProber
	logger    *logp.Logger
}

// AutodiscoverBuilder builds a Jolokia Discovery autodiscover provider, it fails if
// there is some problem with the configuration
func AutodiscoverBuilder(
	beatName string,
	bus bus.Bus,
	uuid uuid.UUID,
	c *config.C,
	keystore keystore.Keystore,
	logger *logp.Logger,
) (autodiscover.Provider, error) {
	errWrap := func(err error) error {
		return fmt.Errorf("error setting up jolokia autodiscover provider: %w", err)
	}

	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, errWrap(err)
	}

	discovery := &Discovery{
		ProviderUUID: uuid,
		Interfaces:   config.Interfaces,
		log:          logger,
	}

	mapper, err := template.NewConfigMapper(config.Templates, keystore, nil, logger)
	if err != nil {
		return nil, errWrap(err)
	}
	if len(mapper.ConditionMaps) == 0 {
		return nil, errWrap(fmt.Errorf("no configs defined for autodiscover provider"))
	}

	builders, err := autodiscover.NewBuilders(config.Builders, nil, nil)
	if err != nil {
		return nil, errWrap(err)
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, errWrap(err)
	}

	return &Provider{
		bus:       bus,
		templates: mapper,
		builders:  builders,
		appenders: appenders,
		discovery: discovery,
		logger:    logger.Named("jolokia"),
	}, nil
}

// Start starts autodiscover provider
func (p *Provider) Start() {
	p.discovery.Start()
	go func() {
		for event := range p.discovery.Events() {
			p.publish(event.BusEvent())
		}
	}()
}

func (p *Provider) publish(event bus.Event) {
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else if config := p.builders.GetConfig(event); config != nil {
		event["config"] = config
	}

	p.appenders.Append(event)
	p.bus.Publish(event)
}

// Stop stops autodiscover provider
func (p *Provider) Stop() {
	p.discovery.Stop()
}

// String returns the name of the provider
func (p *Provider) String() string {
	return "jolokia"
}
