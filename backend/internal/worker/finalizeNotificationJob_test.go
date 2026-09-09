package worker

import "testing"

func TestNotificationJobStatus(t *testing.T) {
	tests := []struct {
		name     string
		counts   deliveryStatusCounts
		status   string
		complete bool
	}{
		{name: "no deliveries", counts: deliveryStatusCounts{}, complete: false},
		{name: "still pending", counts: deliveryStatusCounts{Total: 3, Successful: 2, Unfinished: 1}, complete: false},
		{name: "all successful", counts: deliveryStatusCounts{Total: 3, Successful: 3}, status: "done", complete: true},
		{name: "partially failed", counts: deliveryStatusCounts{Total: 3, Successful: 2, Failed: 1}, status: "partially_failed", complete: true},
		{name: "all failed", counts: deliveryStatusCounts{Total: 3, Failed: 3}, status: "failed", complete: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, complete := notificationJobStatus(test.counts)
			if status != test.status || complete != test.complete {
				t.Fatalf("notificationJobStatus(%+v) = (%q, %t), want (%q, %t)", test.counts, status, complete, test.status, test.complete)
			}
		})
	}
}
