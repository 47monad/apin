package rmqinitr

import (
	"testing"
	"time"
)

func TestNextRetryIntervalDoublesUpToMax(t *testing.T) {
	shell := &Shell{config: &resolvedConfig{
		minRetryInterval: time.Second,
		maxRetryInterval: 4 * time.Second,
	}}

	cases := []struct {
		current time.Duration
		want    time.Duration
	}{
		{time.Second, 2 * time.Second},
		{2 * time.Second, 4 * time.Second},
		{4 * time.Second, 4 * time.Second},
		{3 * time.Second, 4 * time.Second},
	}

	for _, tc := range cases {
		if got := shell.nextRetryInterval(tc.current); got != tc.want {
			t.Errorf("nextRetryInterval(%v) = %v, want %v", tc.current, got, tc.want)
		}
	}
}
