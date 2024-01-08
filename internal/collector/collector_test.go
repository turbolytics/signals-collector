package collector

import (
	"github.com/stretchr/testify/assert"
	"github.com/turbolytics/collector/internal/config"
	"github.com/turbolytics/collector/internal/metrics"
	"testing"
)

func TestCollector_Transform_AddTagsFromConfig(t *testing.T) {
	coll, err := New(&config.Config{
		Metric: config.Metric{
			Tags: []config.Tag{
				{"key1", "val1"},
				{"key2", "val2"},
			},
		},
	})
	assert.NoError(t, err)

	ms := []*metrics.Metric{{
		Tags: make(map[string]string),
	}}

	err = coll.Transform(ms)

	assert.NoError(t, err)
	assert.Equal(t, []*metrics.Metric{{
		Tags: map[string]string{
			"key1": "val1",
			"key2": "val2",
		},
	}}, ms)
}

func TestCollector_Close(t *testing.T) {
	ts := &TestSink{}
	coll := &Collector{
		Config: &config.Config{
			Sinks: map[string]config.Sink{
				"sink1": {
					Sinker: ts,
				},
				"sink2": {
					Sinker: ts,
				},
			},
		},
	}
	err := coll.Close()
	assert.NoError(t, err)
	assert.Equal(t, 2, ts.closes)
}
