package main

import "testing"

func TestValidStatus(t *testing.T) {
	valid := []string{"accepted", "preparing", "ready", "delivered", "cancelled"}
	for _, status := range valid {
		if !validStatus(status) {
			t.Fatalf("expected status %q to be valid", status)
		}
	}
	for _, status := range []string{"pending", "done", "", "unknown"} {
		if validStatus(status) {
			t.Fatalf("expected status %q to be invalid", status)
		}
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		{"accepted", "preparing", true},
		{"accepted", "cancelled", true},
		{"preparing", "ready", true},
		{"ready", "delivered", true},
		{"delivered", "preparing", false},
		{"cancelled", "accepted", false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"_to_"+tt.to, func(t *testing.T) {
			if got := canTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("canTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
