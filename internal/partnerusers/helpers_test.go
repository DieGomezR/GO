package partnerusers

import "testing"

func TestBasePartner(t *testing.T) {
	t.Parallel()

	service := &Service{}
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "removes numeric suffix", input: "suparo_eirl_2277", expected: "suparo_eirl"},
		{name: "removes qa prefix", input: "qaloa_network", expected: "loa_network"},
		{name: "keeps internal underscore", input: "loa_network_1", expected: "loa_network"},
		{name: "plain partner", input: "quality", expected: "quality"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := service.basePartner(tc.input); got != tc.expected {
				t.Fatalf("basePartner(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	service := &Service{}
	got := service.normalize("QA-Loa_Network#1")
	if got != "qaloanetwork1" {
		t.Fatalf("normalize() = %q, want %q", got, "qaloanetwork1")
	}
}
