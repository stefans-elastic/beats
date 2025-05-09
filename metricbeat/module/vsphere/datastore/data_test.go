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

package datastore

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestEventMapping(t *testing.T) {
	var datastoreTest = mo.Datastore{
		Summary: types.DatastoreSummary{
			Name:      "datastore-test",
			Type:      "local",
			Capacity:  5000000,
			FreeSpace: 5000000,
		},
		ManagedEntity: mo.ManagedEntity{
			OverallStatus: "green",
			ExtensibleManagedObject: mo.ExtensibleManagedObject{
				Self: types.ManagedObjectReference{
					Value: "DS_1",
				},
			},
		},
		Host: []types.DatastoreHostMount{},
		Vm: []types.ManagedObjectReference{
			{Type: "VirtualMachine", Value: "vm-test"},
		},
	}

	var metricDataTest = metricData{
		perfMetrics: map[string]interface{}{
			"datastore.read.average":      int64(100),
			"datastore.write.average":     int64(200),
			"disk.capacity.latest":        int64(10000),
			"disk.capacity.usage.average": int64(5000),
			"disk.provisioned.latest":     int64(5000),
		},
		assetNames: assetNames{
			outputHostNames: []string{"DC3_H0"},
			outputVmNames:   []string{"DC3_H0_VM0"},
		},
	}

	outputEvent := (&DataStoreMetricSet{}).mapEvent(datastoreTest, &metricDataTest)
	testEvent := mapstr.M{
		"fstype": "local",
		"id":     "DS_1",
		"status": types.ManagedEntityStatus("green"),
		"name":   "datastore-test",
		"host": mapstr.M{
			"count": 1,
			"names": []string{"DC3_H0"},
		},
		"vm": mapstr.M{
			"count": 1,
			"names": []string{"DC3_H0_VM0"},
		},
		"read": mapstr.M{
			"bytes": int64(102400),
		},
		"write": mapstr.M{
			"bytes": int64(204800),
		},
		"disk": mapstr.M{
			"capacity": mapstr.M{
				"bytes": int64(10240000),
				"usage": mapstr.M{
					"bytes": int64(5120000),
				},
			},
			"provisioned": mapstr.M{
				"bytes": int64(5120000),
			},
		},
		"capacity": mapstr.M{
			"free": mapstr.M{
				"bytes": int64(5000000),
			},
			"total": mapstr.M{
				"bytes": int64(5000000),
			},
			"used": mapstr.M{
				"bytes": int64(0),
				"pct":   float64(0),
			},
		},
	}

	assert.Exactly(t, outputEvent, testEvent)
}
