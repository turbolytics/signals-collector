package config

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSchedule_SetDefaults_Schedule(t *testing.T) {
	conf := &Config{}
	defaults(conf)
	assert.Equal(t, TypeSchedulerStrategyTick, conf.Schedule.Strategy)
}

func TestSchedule_Validate(t *testing.T) {
	h := time.Hour
	c := "* * * * *"

	testCases := []struct {
		name     string
		schedule Schedule
		err      error
	}{
		{
			name:     "must_not_miss_interval_and_cron",
			schedule: Schedule{},
			err:      fmt.Errorf("must set schedule.interval or schedule.cron"),
		},
		{
			name: "must_not_have_interval_and_cron",
			schedule: Schedule{
				Interval: &h,
				Cron:     &c,
			},
			err: fmt.Errorf("must set either schedule.interval or schedule.cro"),
		},
		{
			name: "must_have_valid_strategy",
			schedule: Schedule{
				Cron: &c,
			},
			err: fmt.Errorf("unknown strategy: \"\""),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.schedule.Validate()
			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, tc.err, err)
			}
		})
	}
}
