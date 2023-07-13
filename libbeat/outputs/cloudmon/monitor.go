package cloudmon

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

type monitor struct {
	filter  queryFunc
	counter atomic.Int64
}

func newMonitor(config *MonitorConfig) (*monitor, error) {
	ast, err := NewQueryParser().Parse(config.Query)
	if err != nil {
		return nil, err
	}

	return &monitor{
		filter: newQuery(ast),
	}, nil
}

func mustMonitor(config *MonitorConfig) *monitor {
	m, err := newMonitor(config)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *monitor) Update(e *beat.Event) {
	if m.filter(LogEntry(e.Fields)) {
		m.counter.Inc()
	}
}
