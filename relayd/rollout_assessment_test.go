package main

import (
	"errors"
	"testing"
)

func TestCanaryAssessmentRequiresEnoughTrustworthyTraffic(t *testing.T) {
	cases := []struct {
		name   string
		total  int
		errors int
		logErr error
		want   string
	}{
		{name: "no traffic", total: 0, want: "hold"},
		{name: "below minimum", total: 24, want: "hold"},
		{name: "failed log read", total: 50, logErr: errors.New("unavailable"), want: "hold"},
		{name: "healthy sample", total: 25, errors: 1, want: "promote"},
		{name: "unhealthy sample", total: 25, errors: 2, want: "rollback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canaryAssessment(tc.total, tc.errors, 25, 5, tc.logErr); got != tc.want {
				t.Fatalf("assessment = %s, want %s", got, tc.want)
			}
		})
	}
}
