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
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
)

type Config struct {
	Codec codec.Config `config:"codec"`

	// old pretty settings to use if no codec is configured
	Pretty bool `config:"pretty"`

	BatchSize int

	Monitors []*MonitorConfig `config:"monitors"`
}

type MonitorConfig struct {
	Title     string   `config:"title"`
	Query     string   `config:"query"`
	AggFunc   string   `config:"agg_func"`
	AggArgs   []string `config:"agg_args"`
	AggGroup  []string `config:"agg_group"`
	AggWindow string   `config:"agg_window"`
	Threshold string   `config:"threshold"`
}

var defaultConfig = Config{
	Monitors: []*MonitorConfig{},
}
