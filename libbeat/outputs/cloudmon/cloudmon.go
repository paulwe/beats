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

package cloudmon

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type cloudmon struct {
	log      *logp.Logger
	observer outputs.Observer
	monitors []*monitor
	index    string
}

func init() {
	outputs.RegisterType("cloudmon", makeCloudMon)
}

func makeCloudMon(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *config.C,
) (outputs.Group, error) {
	config := defaultConfig
	err := cfg.Unpack(&config)
	if err != nil {
		return outputs.Fail(err)
	}

	var monitors []*monitor
	for _, mc := range config.Monitors {
		m, err := newMonitor(mc)
		if err != nil {
			return outputs.Fail(fmt.Errorf("cloudmon monitor initialization failed with: %v", err))
		}
		monitors = append(monitors, m)
	}

	c := newCloudMon(beat.Beat, observer, monitors)

	return outputs.Success(config.BatchSize, 0, c)
}

func newCloudMon(index string, observer outputs.Observer, monitors []*monitor) *cloudmon {
	return &cloudmon{
		log:      logp.NewLogger("cloudmon"),
		observer: observer,
		monitors: monitors,
		index:    index,
	}
}

func (c *cloudmon) Close() error { return nil }
func (c *cloudmon) Publish(_ context.Context, batch publisher.Batch) error {
	events := batch.Events()
	c.observer.NewBatch(len(events))

	for i := range events {
		c.updateMonitors(&events[i].Content)
	}

	batch.ACK()

	c.observer.Dropped(0)
	c.observer.Acked(len(events))

	return nil
}

func (c *cloudmon) updateMonitors(e *beat.Event) {
	for _, m := range c.monitors {
		m.Update(e)
	}
}

func (c *cloudmon) String() string {
	return "cloudmon"
}
