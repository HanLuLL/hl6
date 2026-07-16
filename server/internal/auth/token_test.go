package auth

import "testing"

func TestNewRawTokenIsOpaqueAndHashIsStable(t *testing.T) {
	first, err := NewRawToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewRawToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two generated tokens are identical")
	}
	if len(first) < 43 {
		t.Fatalf("token too short: %d", len(first))
	}
	if HashToken(first) != HashToken(first) {
		t.Fatal("token hash is not stable")
	}
	if HashToken(first) == HashToken(second) {
		t.Fatal("distinct tokens have identical hashes")
	}
}

func TestHashTokenUsesSHA256Hex(t *testing.T) {
	got := HashToken("abc")
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
