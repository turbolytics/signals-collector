package internal

import (
	"github.com/turbolytics/collector/internal/metrics"
	"github.com/turbolytics/collector/internal/sinks"
	"github.com/turbolytics/collector/internal/sinks/console"
	"github.com/turbolytics/collector/internal/sinks/http"
	"github.com/turbolytics/collector/internal/sources"
	"github.com/turbolytics/collector/internal/sources/postgres"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

type Metric struct {
	Name string
	Type metrics.Type
}

type Schedule struct {
	Interval time.Duration
	Cron     string
}

type Collector struct {
	Type   string // enum
	Config map[string]any
}

type Sink struct {
	Type   sinks.Type
	Sinker sinks.Sinker
	Config map[string]any
}

type Source struct {
	Type    sources.Type
	Sourcer sources.Sourcer
	Config  map[string]any
}

type ConfigOption func(*Config)

func WithJustValidation(validate bool) ConfigOption {
	return func(c *Config) {
		c.validate = validate
	}
}

type Config struct {
	Name     string
	Metric   Metric
	Schedule Schedule
	Source   Source
	Sinks    map[string]Sink

	// validate will skip initializing network dependencies
	validate bool
}

// initSource initializes the correct source.
func initSource(c *Config) error {
	var s sources.Sourcer
	var err error
	switch c.Source.Type {
	case sources.TypePostgres:
		s, err = postgres.NewFromGenericConfig(
			c.Source.Config,
			c.validate,
		)
	}

	if err != nil {
		return err
	}

	c.Source.Sourcer = s
	return nil
}

// initSinks initializes all the outputs
func initSinks(c *Config) error {
	for k, v := range c.Sinks {
		switch v.Type {
		case sinks.TypeConsole:
			sink, err := console.NewFromGenericConfig(v.Config)
			if err != nil {
				return err
			}
			v.Sinker = sink
			c.Sinks[k] = v
		case sinks.TypeHTTP:
			sink, err := http.NewFromGenericConfig(v.Config)
			if err != nil {
				return err
			}
			v.Sinker = sink
			c.Sinks[k] = v
		}
	}
	return nil
}

func NewConfigFromFile(name string, opts ...ConfigOption) (*Config, error) {
	bs, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return NewConfig(bs, opts...)
}

// NewConfig initializes a config from yaml bytes.
// NewConfig initializes all subtypes as well.
func NewConfig(bs []byte, opts ...ConfigOption) (*Config, error) {
	var conf Config

	for _, opt := range opts {
		opt(&conf)
	}

	if err := yaml.Unmarshal(bs, &conf); err != nil {
		return nil, err
	}

	if err := initSource(&conf); err != nil {
		return nil, err
	}

	if err := initSinks(&conf); err != nil {
		return nil, err
	}
	return &conf, nil
}
