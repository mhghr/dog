package auth

import "testing"

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"09123456789", "+989123456789", false},
		{"۰۹۱۲۳۴۵۶۷۸۹", "+989123456789", false},
		{"+98 912 345 6789", "+989123456789", false},
		{"00989123456789", "+989123456789", false},
		{"9123456789", "+989123456789", false},
		{"+14155552671", "+14155552671", false},
		{"0912-345-6789", "+989123456789", false},
		{"abc", "", true},
		{"+0123", "", true},
		{"", "", true},
	}

	for _, testCase := range cases {
		result, err := NormalizePhone(testCase.input)
		if testCase.wantErr {
			if err == nil {
				t.Errorf("NormalizePhone(%q): expected error, got %q", testCase.input, result)
			}
			continue
		}

		if err != nil {
			t.Errorf("NormalizePhone(%q): unexpected error %v", testCase.input, err)
			continue
		}

		if result != testCase.expected {
			t.Errorf("NormalizePhone(%q) = %q, expected %q", testCase.input, result, testCase.expected)
		}
	}
}

func TestGenerateOTPCode(t *testing.T) {
	seen := map[string]bool{}

	for range 20 {
		code, err := generateOTPCode()
		if err != nil {
			t.Fatalf("generate otp: %v", err)
		}
		if len(code) != otpLength {
			t.Fatalf("unexpected code length %d", len(code))
		}
		for _, char := range code {
			if char < '0' || char > '9' {
				t.Fatalf("non-digit character in code %q", code)
			}
		}
		seen[code] = true
	}

	if len(seen) < 2 {
		t.Fatal("otp codes are not random")
	}
}

func TestHashTokenIsStable(t *testing.T) {
	if hashToken("abc") != hashToken("abc") {
		t.Fatal("hash must be deterministic")
	}
	if hashToken("abc") == hashToken("abd") {
		t.Fatal("different tokens must produce different hashes")
	}
	if len(hashToken("abc")) != 64 {
		t.Fatal("expected sha256 hex length 64")
	}
}
