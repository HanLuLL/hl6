package middleware

import "testing"

func TestParseSessionSubjectAcceptsOnlyPositiveNumericUserIDs(t *testing.T) {
	for _, subject := range []string{"", "0", "external-subject", "1.2", "-1", " 1"} {
		if _, err := parseSessionSubject(subject); err == nil {
			t.Fatalf("invalid subject accepted: %q", subject)
		}
	}
	if userID, err := parseSessionSubject("42"); err != nil || userID != 42 {
		t.Fatalf("got userID=%d err=%v", userID, err)
	}
}

func TestParseSessionVersionRejectsNonIntegralClaims(t *testing.T) {
	for _, claim := range []interface{}{nil, "1", float64(1.5), float64(0), int64(-1)} {
		if _, err := parseSessionVersion(claim); err == nil {
			t.Fatalf("invalid session version accepted: %#v", claim)
		}
	}
	if version, err := parseSessionVersion(uint(3)); err != nil || version != 3 {
		t.Fatalf("got version=%d err=%v", version, err)
	}
}
