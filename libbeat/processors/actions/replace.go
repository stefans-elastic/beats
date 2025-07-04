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
	"regexp"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor/registry"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type replaceString struct {
	config replaceStringConfig
	log    *logp.Logger
}

type replaceStringConfig struct {
	Fields        []replaceConfig `config:"fields"`
	IgnoreMissing bool            `config:"ignore_missing"`
	FailOnError   bool            `config:"fail_on_error"`
}

type replaceConfig struct {
	Field       string         `config:"field" validate:"required"`
	Pattern     *regexp.Regexp `config:"pattern" validate:"required"`
	Replacement *string        `config:"replacement"`
}

func (c replaceConfig) Validate() error {
	if c.Replacement == nil {
		return errors.New("missing replacement")
	}
	return nil
}

func init() {
	processors.RegisterPlugin("replace",
		checks.ConfigChecked(NewReplaceString,
			checks.RequireFields("fields")))

	jsprocessor.RegisterPlugin("Replace", NewReplaceString)
}

// NewReplaceString returns a new replace processor.
func NewReplaceString(c *conf.C, log *logp.Logger) (beat.Processor, error) {
	config := replaceStringConfig{
		IgnoreMissing: false,
		FailOnError:   true,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the replace configuration: %w", err)
	}

	f := &replaceString{
		config: config,
		log:    log.Named("replace"),
	}
	return f, nil
}

func (f *replaceString) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	// Creates a copy of the event to revert in case of failure
	if f.config.FailOnError {
		backup = event.Clone()
	}

	for _, field := range f.config.Fields {
		err := f.replaceField(field.Field, field.Pattern, *field.Replacement, event)
		if err != nil {
			errMsg := fmt.Errorf("Failed to replace fields in processor: %w", err)
			f.log.Debugw(errMsg.Error(), logp.TypeKey, logp.EventType)

			if f.config.FailOnError {
				event = backup
				_, _ = event.PutValue("error.message", errMsg.Error())
				return event, err
			}
		}
	}

	return event, nil
}

func (f *replaceString) replaceField(field string, pattern *regexp.Regexp, replacement string, event *beat.Event) error {
	currentValue, err := event.GetValue(field)
	if err != nil {
		// Ignore ErrKeyNotFound errors
		if f.config.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %w", field, err)
	}

	updatedString := pattern.ReplaceAllString(currentValue.(string), replacement)
	_, err = event.PutValue(field, updatedString)
	if err != nil {
		return fmt.Errorf("could not put value: %s: %v, %w", replacement, currentValue, err)
	}
	return nil
}

func (f *replaceString) String() string {
	return "replace=" + fmt.Sprintf("%+v", f.config.Fields)
}
