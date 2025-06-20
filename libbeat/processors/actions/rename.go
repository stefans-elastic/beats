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

package actions

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor/registry"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type renameFields struct {
	config renameFieldsConfig
	logger *logp.Logger
}

type renameFieldsConfig struct {
	Fields        []fromTo `config:"fields"`
	IgnoreMissing bool     `config:"ignore_missing"`
	FailOnError   bool     `config:"fail_on_error"`
}

type fromTo struct {
	From string `config:"from"`
	To   string `config:"to"`
}

func init() {
	processors.RegisterPlugin("rename",
		checks.ConfigChecked(NewRenameFields,
			checks.RequireFields("fields")))

	jsprocessor.RegisterPlugin("Rename", NewRenameFields)
}

// NewRenameFields returns a new rename processor.
func NewRenameFields(c *conf.C, log *logp.Logger) (beat.Processor, error) {
	config := renameFieldsConfig{
		IgnoreMissing: false,
		FailOnError:   true,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the rename configuration: %w", err)
	}

	f := &renameFields{
		config: config,
		logger: log.Named("rename"),
	}
	return f, nil
}

func (f *renameFields) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	// Creates a copy of the event to revert in case of failure
	if f.config.FailOnError {
		backup = event.Clone()
	}

	for _, field := range f.config.Fields {
		err := f.renameField(field.From, field.To, event)
		if err != nil {
			errMsg := fmt.Errorf("Failed to rename fields in processor: %w", err)
			f.logger.Debugw(errMsg.Error(), logp.TypeKey, logp.EventType)

			if f.config.FailOnError {
				event = backup
				_, _ = event.PutValue("error.message", errMsg.Error())
				return event, err
			}
		}
	}

	return event, nil
}

func (f *renameFields) renameField(from string, to string, event *beat.Event) error {
	// Fields cannot be overwritten. Either the target field has to be dropped first or renamed first
	_, err := event.GetValue(to)
	if err == nil {
		return fmt.Errorf("target field %s already exists, drop or rename this field first", to)
	}

	value, err := event.GetValue(from)
	if err != nil {
		// Ignore ErrKeyNotFound errors
		if f.config.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %w", from, err)
	}

	// Deletion must happen first to support cases where a becomes a.b
	err = event.Delete(from)
	if err != nil {
		return fmt.Errorf("could not delete key: %s,  %w", from, err)
	}

	_, err = event.PutValue(to, value)
	if err != nil {
		return fmt.Errorf("could not put value: %s: %v, %w", to, value, err)
	}
	return nil
}

func (f *renameFields) String() string {
	return "rename=" + fmt.Sprintf("%+v", f.config.Fields)
}
