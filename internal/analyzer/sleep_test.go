package analyzer

import (
	"testing"
	"time"
)

func TestAssessDrain(t *testing.T) {
	tests := []struct {
		name          string
		drainPerHour  float64
		wantAssessment string
	}{
		{
			name:          "normal drain",
			drainPerHour:  0.5,
			wantAssessment: "Normal",
		},
		{
			name:          "exactly at normal threshold",
			drainPerHour:  1.0,
			wantAssessment: "High",
		},
		{
			name:          "high drain",
			drainPerHour:  2.0,
			wantAssessment: "High",
		},
		{
			name:          "exactly at critical threshold",
			drainPerHour:  3.0,
			wantAssessment: "Critical",
		},
		{
			name:          "critical drain",
			drainPerHour:  5.0,
			wantAssessment: "Critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assessDrain(tt.drainPerHour)
			if got != tt.wantAssessment {
				t.Errorf("assessDrain(%v) = %v, want %v", tt.drainPerHour, got, tt.wantAssessment)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     time.Duration
	}{
		{
			name:     "positive duration",
			duration: 5 * time.Minute,
			want:     5 * time.Minute,
		},
		{
			name:     "negative duration",
			duration: -5 * time.Minute,
			want:     5 * time.Minute,
		},
		{
			name:     "zero duration",
			duration: 0,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := abs(tt.duration)
			if got != tt.want {
				t.Errorf("abs(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}
