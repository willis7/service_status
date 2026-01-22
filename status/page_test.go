package status

import (
	"html/template"
	"testing"
)

func TestStatusHTML(t *testing.T) {
	tt := []struct {
		name     string
		input    string
		expected template.HTML
	}{
		{name: "danger status", input: "danger", expected: "danger"},
		{name: "degraded status", input: "degraded", expected: "degraded"},
		{name: "success status", input: "success", expected: "success"},
		{name: "maintenance status", input: "maintenance", expected: "maintenance"},
		{name: "empty status defaults to success", input: "", expected: "success"},
		{name: "unknown status defaults to success", input: "unknown", expected: "success"},
		{name: "warning defaults to success", input: "warning", expected: "success"},
		{name: "arbitrary string defaults to success", input: "<script>alert('xss')</script>", expected: "success"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if got := StatusHTML(tc.input); got != tc.expected {
				t.Errorf("StatusHTML(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}
