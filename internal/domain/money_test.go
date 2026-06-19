package domain

import "testing"

func TestParseMoney_ValidAmounts(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"12.34", 1234},
		{"0.01", 1},
		{"100", 10000},
	}

	for _, c := range cases {
		m, err := ParseMoney(c.input)
		if err != nil {
			t.Fatalf("expected no error for %q: %v", c.input, err)
		}
		if m.Cents != c.expected {
			t.Fatalf("expected %d cents for %q, got %d", c.expected, c.input, m.Cents)
		}
	}
}

func TestParseMoney_InvalidAmounts(t *testing.T) {
	cases := []string{"abc", "-5.00", "0.001", ""}

	for _, input := range cases {
		if _, err := ParseMoney(input); err == nil {
			t.Fatalf("expected error for invalid amount %q", input)
		}
	}
}
