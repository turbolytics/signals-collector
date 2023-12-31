package internal

import (
	"bytes"
	"context"
	"fmt"
	"github.com/turbolytics/collector/internal/metrics"
	"github.com/turbolytics/collector/internal/sinks"
	"github.com/turbolytics/collector/internal/sinks/console"
	"github.com/turbolytics/collector/internal/sinks/file"
	"github.com/turbolytics/collector/internal/sinks/http"
	"github.com/turbolytics/collector/internal/sinks/kafka"
	"github.com/turbolytics/collector/internal/sources"
	"github.com/turbolytics/collector/internal/sources/mongodb"
	"github.com/turbolytics/collector/internal/sources/postgres"
	"github.com/turbolytics/collector/internal/sources/prometheus"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"text/template"
	"time"
)

type Tag struct {
	Key   string
	Value string
}

type Metric struct {
	Name string
	Type metrics.Type
	// Grain specifies the aggregation of this metric. It will be used to calculate the
	// grain datetime.
	Grain *time.Duration
	Tags  []Tag
}

type Schedule struct {
	Interval *time.Duration
	Cron     *string
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

func WithJustConfigValidation(validate bool) ConfigOption {
	return func(c *Config) {
		c.validate = validate
	}
}

func WithConfigLogger(l *zap.Logger) ConfigOption {
	return func(c *Config) {
		c.logger = l
	}
}

type Config struct {
	Name     string
	Metric   Metric
	Schedule Schedule
	Source   Source
	Sinks    map[string]Sink

	logger *zap.Logger
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
	case sources.TypeMongoDB:
		s, err = mongodb.NewFromGenericConfig(
			context.TODO(),
			c.Source.Config,
			c.validate,
		)
	case sources.TypePrometheus:
		s, err = prometheus.NewFromGenericConfig(
			c.Source.Config,
			prometheus.WithLogger(c.logger),
		)
	default:
		return fmt.Errorf("source type: %q unknown", c.Source.Type)
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
		case sinks.TypeKafka:
			sink, err := kafka.NewFromGenericConfig(v.Config)
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
		case sinks.TypeFile:
			sink, err := file.NewFromGenericConfig(
				v.Config,
				c.validate,
			)
			if err != nil {
				return err
			}
			v.Sinker = sink
			c.Sinks[k] = v
		}
	}
	return nil
}

func parseTemplate(bs []byte) ([]byte, error) {
	funcMap := template.FuncMap{
		"getEnv": func(key string) string {
			return os.Getenv(key)
		},
		"getEnvOrDefault": func(key string, d string) string {
			envVal := os.Getenv(key)
			if envVal == "" {
				return d
			}

			return envVal
		},
	}
	t, err := template.New("config").Funcs(funcMap).Parse(string(bs))
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := t.Execute(&out, nil); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func NewConfigsFromDir(dirname string, opts ...ConfigOption) ([]*Config, error) {
	files, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	var configs []*Config

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		n := path.Join(dirname, f.Name())
		c, err := NewConfigFromFile(n, opts...)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func NewConfigFromFile(name string, opts ...ConfigOption) (*Config, error) {
	bs, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return NewConfig(bs, opts...)
}

type validator func(Config) error

func validateMetric(c Config) error {
	if c.Schedule.Interval != nil && (c.Schedule.Interval.Nanoseconds() != c.Metric.Grain.Nanoseconds()) {
		return fmt.Errorf("'schedule.interval' should match 'metric.grain'")
	}
	return nil
}

func validateSchedule(c Config) error {
	if c.Schedule.Interval == nil && c.Schedule.Cron == nil {
		return fmt.Errorf("must set schedule.interval or schedule.cron")
	}

	if c.Schedule.Interval != nil && c.Schedule.Cron != nil {
		return fmt.Errorf("must set either schedule.interval or schedule.cron")
	}

	return nil
}

func validate(c Config) error {
	validators := []validator{
		validateSchedule,
		validateMetric,
	}

	for _, vFn := range validators {
		if err := vFn(c); err != nil {
			return err
		}
	}
	return nil
}

// NewConfig initializes a config from yaml bytes.
// NewConfig initializes all subtypes as well.
func NewConfig(raw []byte, opts ...ConfigOption) (*Config, error) {
	var conf Config

	for _, opt := range opts {
		opt(&conf)
	}

	bs, err := parseTemplate(raw)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(bs, &conf); err != nil {
		return nil, err
	}

	if err := validate(conf); err != nil {
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
