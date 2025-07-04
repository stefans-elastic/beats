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

//go:build mage

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"
	filebeat "github.com/elastic/beats/v7/filebeat/scripts/mage"

	//mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	//mage:import generate
	_ "github.com/elastic/beats/v7/filebeat/scripts/mage/generate"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	//mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/test"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/docker"
)

func init() {
	common.RegisterCheckDeps(Update)
	test.RegisterDeps(IntegTest)

	devtools.BeatDescription = "Filebeat sends log files to Logstash or directly to Elasticsearch."
}

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// BuildSystemTestBinary builds a binary instrumented for use with Python system tests.
func BuildSystemTestBinary() error {
	return devtools.BuildSystemTestBinary()
}

// GolangCrossBuild builds the Beat binary inside the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return filebeat.GolangCrossBuild()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return filebeat.CrossBuild()
}

// AssembleDarwinUniversal merges the darwin/amd64 and darwin/arm64 into a single
// universal binary using `lipo`. It assumes the darwin/amd64 and darwin/arm64
// were built and only performs the merge.
func AssembleDarwinUniversal() error {
	return build.AssembleDarwinUniversal()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatOSSPackaging()
	devtools.PackageKibanaDashboardsFromBuildDir()
	filebeat.CustomizePackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// Package packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	fmt.Println(">> Ironbank: this module is not subscribed to the IronBank releases.")
	return nil
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(devtools.WithModules(), devtools.WithModulesD())
}

// Update is an alias for executing fields, dashboards, config, includes.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config, GenerateModuleIncludeListGo, fieldDocs,
		filebeat.CollectDocs,
		filebeat.PrepareModulePackagingOSS)
}

// Config generates both the short/reference/docker configs and populates the
// modules.d directory.
func Config() {
	mg.Deps(devtools.GenerateDirModulesD, configYML)
	mg.SerialDeps(devtools.ValidateDirModulesD, devtools.ValidateDirModulesDDatasetsDisabled)
}

func configYML() error {
	return devtools.Config(devtools.AllConfigTypes, filebeat.OSSConfigFileParams(), ".")
}

// GenerateModuleIncludeListGo generates include/list.go with imports for inputs.
func GenerateModuleIncludeListGo() error {
	options := devtools.DefaultIncludeListOptions()
	options.ImportDirs = []string{"autodiscover", "autodiscover/**/*", "input", "input/*", "processor/*"}
	return devtools.GenerateIncludeListGo(options)
}

// Fields generates fields.yml and fields.go files for the Beat.
func Fields() {
	mg.Deps(libbeatAndFilebeatCommonFieldsGo, moduleFieldsGo)
	mg.Deps(fieldsYML)
}

// libbeatAndFilebeatCommonFieldsGo generates a fields.go containing both
// libbeat and filebeat's common fields.
func libbeatAndFilebeatCommonFieldsGo() error {
	if err := devtools.GenerateFieldsYAML(); err != nil {
		return err
	}
	return devtools.GenerateAllInOneFieldsGo()
}

// moduleFieldsGo generates a fields.go for each module.
func moduleFieldsGo() error {
	return devtools.GenerateModuleFieldsGo("module")
}

// fieldsYML generates the fields.yml file containing all fields.
func fieldsYML() error {
	return devtools.GenerateFieldsYAML("module")
}

// fieldDocs generates docs/fields.asciidoc containing all fields
// (including x-pack).
func fieldDocs() error {
	inputs := []string{
		devtools.OSSBeatDir("module"),
		devtools.XPackBeatDir("module"),
		devtools.OSSBeatDir("input"),
		devtools.XPackBeatDir("input"),
		devtools.XPackBeatDir("processors"),
	}
	output := devtools.CreateDir("build/fields/fields.all.yml")
	if err := devtools.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return devtools.Docs.FieldDocs(output)
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return devtools.KibanaDashboards("module")
}

// ExportDashboard exports a dashboard and writes it into the correct directory.
//
// Required environment variables:
// - MODULE: Name of the module
// - ID:     Dashboard id
func ExportDashboard() error {
	return devtools.ExportDashboard()
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	mg.SerialDeps(GoIntegTest, PythonIntegTest)
}

// GoIntegTest starts the docker containers and executes the Go integration tests.
func GoIntegTest(ctx context.Context) error {
	mg.Deps(BuildSystemTestBinary)
	return devtools.GoIntegTestFromHost(ctx, devtools.DefaultGoTestIntegrationFromHostArgs())
}

// GoFIPSOnlyIntegTest starts the docker containers and executes the Go integration tests with GODEBUG=fips140=only set.
func GoFIPSOnlyIntegTest(ctx context.Context) error {
	mg.Deps(BuildSystemTestBinary)
	return devtools.GoIntegTestFromHost(ctx, devtools.FIPSOnlyGoTestIntegrationFromHostArgs())
}

// PythonIntegTest starts the docker containers and executes the Python integration tests.
func PythonIntegTest(ctx context.Context) error {
	mg.Deps(Fields, Dashboards, devtools.BuildSystemTestBinary)
	return devtools.PythonIntegTestFromHost(devtools.DefaultPythonTestIntegrationFromHostArgs())
}
