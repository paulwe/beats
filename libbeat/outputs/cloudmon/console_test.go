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

//go:build !integration

package cloudmon

import (
	"context"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/require"
)

func TestCloudMonOutput(t *testing.T) {
	tests := []struct {
		title    string
		monitors []*monitor
		events   []beat.Event
		expected int64
	}{
		{
			"match a filter...",
			[]*monitor{
				mustMonitor(&MonitorConfig{
					Title:     `Egress Server TWIRP errors in logs`,
					Query:     `service:(linode-livekit-cloud-media-production-v1 OR livekit-cloud-media-production-v1 OR cloud-media) env:production 'no stream matches subject'`,
					AggFunc:   "count",
					AggWindow: "5m",
					Threshold: `> 0`,
				}),
			},
			[]beat.Event{
				{Fields: event("service", "linode-livekit-cloud-media-production-v1", "env", "production", "message", "no stream matches subject")},
			},
			1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.title, func(t *testing.T) {
			batch := outest.NewBatch(test.events...)
			n, err := run(test.monitors, batch)
			require.NoError(t, err)

			require.Equal(t, test.expected, n)
		})
	}
}

func run(monitors []*monitor, batches ...publisher.Batch) (int64, error) {
	c := newCloudMon("test", outputs.NewNilObserver(), monitors)
	for _, b := range batches {
		if err := c.Publish(context.Background(), b); err != nil {
			return 0, err
		}
	}
	return c.monitors[0].counter.Load(), nil
}

func event(kv ...string) mapstr.M {
	m := mapstr.M{}
	for i := 1; i < len(kv); i += 2 {
		m[kv[i-1]] = kv[i]
	}
	return m
}
