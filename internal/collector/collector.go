package collector

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/turbolytics/collector/internal/config"
	"github.com/turbolytics/collector/internal/metrics"
	"github.com/turbolytics/collector/internal/obs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"time"
)

var meter = otel.Meter("signals-collector")

type Collector struct {
	logger *zap.Logger
	Config *config.Config
}

func (c *Collector) Close() error {
	for _, s := range c.Config.Sinks {
		s.Sinker.Close()
	}
	return nil
}

func (c *Collector) Transform(ms []*metrics.Metric) error {
	for _, m := range ms {
		m.Name = c.Config.Metric.Name
		m.Type = c.Config.Metric.Type

		// enrich with tags
		// should these be copied?
		for _, t := range c.Config.Metric.Tags {
			m.Tags[t.Key] = t.Value
		}

	}
	return nil
}

func (c *Collector) Source(ctx context.Context) (ms []*metrics.Metric, err error) {
	start := time.Now().UTC()

	histogram, _ := meter.Float64Histogram(
		"collector.source.duration",
		metric.WithUnit("s"),
	)

	defer func() {
		duration := time.Since(start)

		histogram.Record(ctx, duration.Seconds(), metric.WithAttributeSet(
			attribute.NewSet(
				attribute.String("result.status_code", obs.ErrToStatus(err)),
				attribute.String("source.type", string(c.Config.Source.Type)),
			),
		))

		meter.Int64ObservableGauge(
			"collector.source.metrics.total",
			metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(int64(len(ms)), metric.WithAttributeSet(
					attribute.NewSet(
						attribute.String("source.type", string(c.Config.Source.Type)),
					),
				))
				return nil
			}),
		)

	}()

	ms, err = c.Config.Source.Sourcer.Source(ctx)
	if err != nil {
		return nil, err
	}

	return ms, nil
}

func (c *Collector) Sink(ctx context.Context, metrics []*metrics.Metric) error {

	histogram, _ := meter.Float64Histogram(
		"collector.sink.duration",
		metric.WithUnit("s"),
	)
	// add tags from config

	// need to add a serializer
	for _, m := range metrics {
		bs, err := json.Marshal(m)
		if err != nil {
			return err
		}
		for _, s := range c.Config.Sinks {
			start := time.Now().UTC()

			_, err := s.Sinker.Write(bs)

			duration := time.Since(start)
			histogram.Record(ctx, duration.Seconds(), metric.WithAttributeSet(
				attribute.NewSet(
					attribute.String("result.status_code", obs.ErrToStatus(err)),
					attribute.String("sink.name", string(s.Type)),
				),
			))

			if err != nil {
				return err
			}

		}
	}
	return nil
}

// InvokeHandleError will log any Invoke errors and not return them.
// Useful for async scheduling.
func (c *Collector) InvokeHandleError(ctx context.Context) {
	_, err := c.Invoke(ctx)
	if err != nil {
		c.logger.Error(err.Error())
	}
}

func (c *Collector) Invoke(ctx context.Context) (ms []*metrics.Metric, err error) {
	start := time.Now().UTC()

	histogram, _ := meter.Float64Histogram(
		"collector.invoke.duration",
		metric.WithUnit("s"),
	)

	counter, _ := meter.Int64Counter(
		"collector.invoke.count",
	)

	defer func() {
		duration := time.Since(start)

		counter.Add(ctx, 1, metric.WithAttributeSet(
			attribute.NewSet(
				attribute.String("result.status_code", obs.ErrToStatus(err)),
				attribute.String("collector.name", c.Config.Name),
			),
		))

		histogram.Record(ctx, duration.Seconds(), metric.WithAttributeSet(
			attribute.NewSet(
				attribute.String("result.status_code", obs.ErrToStatus(err)),
				attribute.String("collector.name", c.Config.Name),
			),
		))
	}()

	id := uuid.New()
	c.logger.Info(
		"collector.Invoke",
		zap.String("id", id.String()),
		zap.String("name", c.Config.Name),
	)
	ctx = context.WithValue(ctx, "id", id)
	ms, err = c.Source(ctx)
	if err != nil {
		return nil, err
	}

	// only sink if metrics are present:
	if len(ms) == 0 {
		c.logger.Warn(
			"collector.Invoke",
			zap.String("msg", "no metrics found"),
			zap.String("id", id.String()),
			zap.String("name", c.Config.Name),
		)
		return ms, err
	}

	if err = c.Transform(ms); err != nil {
		return ms, err
	}

	if err = c.Sink(ctx, ms); err != nil {
		return ms, err
	}

	return ms, err
}

type Option func(*Collector)

func WithLogger(l *zap.Logger) Option {
	return func(c *Collector) {
		c.logger = l
	}
}

func New(config *config.Config, opts ...Option) (*Collector, error) {
	c := &Collector{
		Config: config,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func NewFromConfigs(configs []*config.Config, opts ...Option) ([]*Collector, error) {
	var cs []*Collector
	for _, config := range configs {
		coll, err := New(config, opts...)
		if err != nil {
			return nil, err
		}
		cs = append(cs, coll)
	}
	return cs, nil
}
