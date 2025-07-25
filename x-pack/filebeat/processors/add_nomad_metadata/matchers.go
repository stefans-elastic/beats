// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/elastic/beats/v7/x-pack/libbeat/processors/add_nomad_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// LogPathMatcherName is the name of LogPathMatcher
const LogPathMatcherName = "logs_path"

const (
	pathSeparator = string(os.PathSeparator)
	allocIDRegex  = "[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}"
)

// const allocIDTypeRegex = "([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}).*(stdout|stderr)"

// InitializeModule initializes all the options for the `add_nomad_metadata` process for filebeat.
func InitializeModule() {
	add_nomad_metadata.Indexing.AddMatcher(LogPathMatcherName, newLogsPathMatcher)
	cfg := conf.NewConfig()

	// Add a container indexer config by default.
	add_nomad_metadata.Indexing.AddDefaultIndexerConfig(add_nomad_metadata.AllocationNameIndexerName, *cfg)
	add_nomad_metadata.Indexing.AddDefaultIndexerConfig(add_nomad_metadata.AllocationUUIDIndexerName, *cfg)

	// Add a log path matcher which can extract container ID from the "source" field.
	add_nomad_metadata.Indexing.AddDefaultMatcherConfig(LogPathMatcherName, *cfg)
}

// LogPathMatcher matches an event by the UUID in the path
type LogPathMatcher struct {
	LogsPath     string
	allocIDRegex *regexp.Regexp
	logger       *logp.Logger
}

func newLogsPathMatcher(cfg conf.C, logger *logp.Logger) (add_nomad_metadata.Matcher, error) {
	config := struct {
		LogsPath string `config:"logs_path"`
	}{
		LogsPath: defaultLogPath(),
	}

	err := cfg.Unpack(&config)
	if err != nil || config.LogsPath == "" {
		return nil, fmt.Errorf("fail to unpack the `logs_path` configuration: %s", err)
	}

	logPath := config.LogsPath
	if logPath[len(logPath)-1:] != pathSeparator {
		logPath = logPath + pathSeparator
	}

	logger.Named("nomad").Debugf("logs_path matcher log path: %s", logPath)

	return &LogPathMatcher{
		LogsPath:     logPath,
		allocIDRegex: regexp.MustCompile(allocIDRegex),
		logger:       logger,
	}, nil
}

// MetadataIndex returns the index key to be used for enriching the event with the proper metadata
// which is the allocation id from the event `log.file.path` field
func (m *LogPathMatcher) MetadataIndex(event mapstr.M) string {
	value, err := event.GetValue("log.file.path")

	if err == nil {
		path := value.(string)
		m.logger.Named("nomad").Debugf("Incoming log.file.path value: %s", path)

		if !strings.Contains(path, m.LogsPath) {
			m.logger.Named("nomad").Debugf("Error extracting allocation id - source value does not contain matcher's logs_path '%s'.", m.LogsPath)
			return ""
		}

		// `log.file.path` looks something like:
		// /appdata/nomad/alloc/389d1bc4-fae4-6956-9f66-6df59a0f11f0/alloc/logs/app-name.stderr.0
		// /appdata/nomad/alloc/18e5cd07-03bb-be76-35e5-39c799d369e6/alloc/logs/app-name.stdout.0

		if !m.allocIDRegex.MatchString(path) {
			m.logger.Named("nomad").Debugf("Error extracting allocation id - source value doesn't contain a valid UUID '%s'.", path)
			return ""
		}

		return m.allocIDRegex.FindString(path)
	}

	return ""
}

func defaultLogPath() string {
	return "/var/lib/nomad"
}
