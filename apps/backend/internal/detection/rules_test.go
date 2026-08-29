package detection

import "testing"

func TestMatches(t *testing.T) {
	tests := []struct {
		name   string
		rule   Rule
		signal Signal
		want   bool
	}{
		{name: "absolute positive", rule: Rule{Enabled: true, Metric: "any", Operator: "abs_z_gte", Threshold: 6}, signal: Signal{Metric: "power_dbm", ZScore: 7}, want: true},
		{name: "absolute negative", rule: Rule{Enabled: true, Metric: "frequency_hz", Operator: "abs_z_gte", Threshold: 6}, signal: Signal{Metric: "frequency_hz", ZScore: -7}, want: true},
		{name: "metric mismatch", rule: Rule{Enabled: true, Metric: "power_dbm", Operator: "abs_z_gte", Threshold: 6}, signal: Signal{Metric: "frequency_hz", ZScore: 7}, want: false},
		{name: "disabled", rule: Rule{Enabled: false, Metric: "any", Operator: "abs_z_gte", Threshold: 1}, signal: Signal{Metric: "power_dbm", ZScore: 9}, want: false},
		{name: "positive direction", rule: Rule{Enabled: true, Metric: "any", Operator: "z_gte", Threshold: 5}, signal: Signal{Metric: "power_dbm", ZScore: 5.1}, want: true},
		{name: "negative direction", rule: Rule{Enabled: true, Metric: "any", Operator: "z_lte", Threshold: 5}, signal: Signal{Metric: "power_dbm", ZScore: -5.1}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Matches(test.rule, test.signal); got != test.want {
				t.Fatalf("Matches() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestHigherSeverity(t *testing.T) {
	if got := HigherSeverity("high", "critical"); got != "critical" {
		t.Fatalf("HigherSeverity() = %q, want critical", got)
	}
	if got := HigherSeverity("medium", "low"); got != "medium" {
		t.Fatalf("HigherSeverity() = %q, want medium", got)
	}
}
