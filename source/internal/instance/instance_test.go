package instance

import "testing"

func TestGenerateSlug(t *testing.T) {
	cases := map[string]string{
		"ATM 10":              "atm-10",
		"DeceasedCraft 5":     "deceasedcraft-5",
		"Dungeons & Dragons":  "dungeons-dragons",
		"My Server (2026)":    "my-server-2026",
		"  Spaced   Out  ":    "spaced-out",
		"UPPER lower MiXeD":   "upper-lower-mixed",
		"---weird---input---": "weird-input",
	}
	for in, want := range cases {
		if got := GenerateSlug(in); got != want {
			t.Errorf("GenerateSlug(%q)=%q want %q", in, got, want)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	good := []string{"a", "atm-10", "dusk-till-dawn", "x1-y2-z3"}
	for _, s := range good {
		if err := ValidateSlug(s); err != nil {
			t.Errorf("ValidateSlug(%q) unexpected error: %v", s, err)
		}
	}
	bad := []string{"", ".", "..", "a/b", `a\b`, "/abs", ".hidden", "UPPER", "has space", "trailing-", "-leading",
		"waaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaay-too-long-over-64-characters"}
	for _, s := range bad {
		if err := ValidateSlug(s); err == nil {
			t.Errorf("ValidateSlug(%q) expected error, got nil", s)
		}
	}
}
