package metrics

import (
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"time"
)

type Type string

const (
	TypeCount Type = "COUNT"
	TypeGauge Type = "GAUGE"
)

type MetricOption func(metric *Metric)

type Metric struct {
	UUID      string            `json:"uuid"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Type      Type              `json:"type"`
	Tags      map[string]string `json:"tags"`
	Timestamp time.Time         `json:"timestamp"`
}

func New(opts ...MetricOption) Metric {
	m := Metric{
		UUID:      uuid.New().String(),
		Tags:      make(map[string]string),
		Timestamp: time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func MapsToMetrics(results []map[string]any) ([]*Metric, error) {
	var ms []*Metric
	for _, r := range results {
		val, ok := r["value"]
		if !ok {
			return nil, fmt.Errorf("each row must contain a %q key", "value")
		}

		m := New()

		switch v := val.(type) {
		case int:
			m.Value = float64(v)
		case int32:
			m.Value = float64(v)
		case int64:
			m.Value = float64(v)
		case string:
			tv, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("unable to parse string to float: %q", v)
			}
			m.Value = tv
		}
		delete(r, "value")
		for k, v := range r {
			m.Tags[k] = v.(string)
		}
		ms = append(ms, &m)
	}
	return ms, nil
}
