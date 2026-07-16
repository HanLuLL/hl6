package auth

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestArgon2idPasswordVerification(t *testing.T) {
	peppers := PepperSet{
		CurrentID: "v1",
		Current:   []byte("test-pepper-current"),
	}

	encoded, err := HashPassword("correct horse battery staple", peppers)
	if err != nil {
		t.Fatal(err)
	}

	valid, needsRehash, err := VerifyPassword("correct horse battery staple", encoded, peppers)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("correct password rejected")
	}
	if needsRehash {
		t.Fatal("fresh hash unexpectedly requires rehash")
	}

	valid, _, err = VerifyPassword("wrong password", encoded, peppers)
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("wrong password accepted")
	}
}

func TestArgon2idPasswordVerificationRequestsRehashForPreviousPepper(t *testing.T) {
	oldPeppers := PepperSet{
		CurrentID: "v1",
		Current:   []byte("test-pepper-old"),
	}
	encoded, err := HashPassword("correct horse battery staple", oldPeppers)
	if err != nil {
		t.Fatal(err)
	}

	rotatedPeppers := PepperSet{
		CurrentID:  "v2",
		Current:    []byte("test-pepper-new"),
		PreviousID: "v1",
		Previous:   []byte("test-pepper-old"),
	}
	valid, needsRehash, err := VerifyPassword("correct horse battery staple", encoded, rotatedPeppers)
	if err != nil {
		t.Fatal(err)
	}
	if !valid || !needsRehash {
		t.Fatalf("got valid=%t needsRehash=%t, want true true", valid, needsRehash)
	}
}

func TestHashPasswordRejectsUnsafePepperID(t *testing.T) {
	_, err := HashPassword("correct horse battery staple", PepperSet{
		CurrentID: "v1,pepper=other",
		Current:   []byte("test-pepper-current"),
	})
	if err == nil {
		t.Fatal("unsafe pepper ID accepted")
	}
}

func TestVerifyPasswordRejectsMalformedPHCInput(t *testing.T) {
	peppers := PepperSet{
		CurrentID: "v1",
		Current:   []byte("test-pepper-current"),
	}
	encoded, err := HashPassword("correct horse battery staple", peppers)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		t.Fatalf("unexpected PHC part count: %d", len(parts))
	}

	tooLongSalt := base64.RawStdEncoding.EncodeToString(bytes.Repeat([]byte{1}, passwordSaltLength+1))
	tooShortHash := base64.RawStdEncoding.EncodeToString(bytes.Repeat([]byte{2}, passwordKeyLength-1))

	for _, test := range []struct {
		name    string
		encoded string
	}{
		{name: "not phc", encoded: "not-a-phc-string"},
		{name: "wrong algorithm", encoded: strings.Replace(encoded, "$argon2id$", "$bcrypt$", 1)},
		{name: "oversized salt", encoded: strings.Join([]string{"", parts[1], parts[2], parts[3], tooLongSalt, parts[5]}, "$")},
		{name: "short hash", encoded: strings.Join([]string{"", parts[1], parts[2], parts[3], parts[4], tooShortHash}, "$")},
		{name: "unsafe pepper id", encoded: strings.Replace(encoded, "pepper=v1", "pepper=../../v1", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid, needsRehash, err := VerifyPassword("correct horse battery staple", test.encoded, peppers)
			if err == nil {
				t.Fatal("malformed PHC input accepted")
			}
			if valid || needsRehash {
				t.Fatalf("got valid=%t needsRehash=%t, want false false", valid, needsRehash)
			}
		})
	}
}
