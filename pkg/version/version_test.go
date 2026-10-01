package version

import (
	"testing"
)

func TestParseAndNormalize(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectMajor int
		expectMinor int
		expectPatch int
		expectPre   string
		expectErr   bool
	}{
		{input: "v0.2.0", expected: "v0.2.0", expectMajor: 0, expectMinor: 2, expectPatch: 0},
		{input: "0.2.0", expected: "v0.2.0", expectMajor: 0, expectMinor: 2, expectPatch: 0},
		{input: "1.0", expected: "v1.0.0", expectMajor: 1, expectMinor: 0, expectPatch: 0},
		{input: "v1", expected: "v1.0.0", expectMajor: 1, expectMinor: 0, expectPatch: 0},
		{input: "v1.2.3-beta.1", expected: "v1.2.3-beta.1", expectMajor: 1, expectMinor: 2, expectPatch: 3, expectPre: "beta.1"},
		{input: "", expected: "v0.0.0", expectMajor: 0, expectMinor: 0, expectPatch: 0},
		{input: "invalid", expectErr: true},
	}

	for _, tc := range tests {
		sv, err := Parse(tc.input)
		if tc.expectErr {
			if err == nil {
				t.Errorf("for input %q: expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("for input %q: unexpected error: %v", tc.input, err)
		}
		if sv.String() != tc.expected {
			t.Errorf("for input %q: expected %q, got %q", tc.input, tc.expected, sv.String())
		}
		if sv.Major != tc.expectMajor || sv.Minor != tc.expectMinor || sv.Patch != tc.expectPatch || sv.Pre != tc.expectPre {
			t.Errorf("for input %q: mismatch fields %+v", tc.input, sv)
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"v0.1.0", "v0.2.0", -1},
		{"v0.2.0", "v0.1.0", 1},
		{"v0.2.0", "v0.2.0", 0},
		{"0.2.0", "v0.2.0", 0},
		{"v0.2.0", "0.2.1", -1},
		{"v1.0.0", "v0.9.9", 1},
		{"v0.2.0-beta", "v0.2.0", -1},
		{"v0.2.0", "v0.2.0-beta", 1},
		{"v0.2.0-alpha", "v0.2.0-beta", -1},
	}

	for _, tc := range tests {
		res := Compare(tc.v1, tc.v2)
		if res != tc.expected {
			t.Errorf("Compare(%q, %q): expected %d, got %d", tc.v1, tc.v2, tc.expected, res)
		}
	}
}
