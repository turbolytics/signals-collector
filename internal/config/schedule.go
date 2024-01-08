package config

import (
	"fmt"
	"time"
)

type TypeSchedulerStrategy string

const (
	TypeSchedulerStrategyStateful TypeSchedulerStrategy = "stateful"
	TypeSchedulerStrategyTick     TypeSchedulerStrategy = "tick"
)

type Schedule struct {
	Interval *time.Duration
	Cron     *string
	Strategy TypeSchedulerStrategy
}

func (s Schedule) Validate() error {
	if s.Interval == nil && s.Cron == nil {
		return fmt.Errorf("must set schedule.interval or schedule.cron")
	}

	if s.Interval != nil && s.Cron != nil {
		return fmt.Errorf("must set either schedule.interval or schedule.cron")
	}

	vs := map[TypeSchedulerStrategy]struct{}{
		TypeSchedulerStrategyTick:     {},
		TypeSchedulerStrategyStateful: {},
	}

	if _, ok := vs[s.Strategy]; !ok {
		return fmt.Errorf("unknown strategy: %q", s.Strategy)
	}

	return nil
}

func (s *Schedule) SetDefaults() {
	if s.Strategy == "" {
		s.Strategy = TypeSchedulerStrategyTick
	}
}
