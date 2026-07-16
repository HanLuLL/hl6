package handler

import "testing"

func TestValidateRestoreConfirmationRequiresExactPhrase(t *testing.T) {
	for _, value := range []string{"", "RESTORE", "restore database", "RESTORE DATABASE ", "RESTORE  DATABASE"} {
		if validateRestoreConfirmation(value) {
			t.Fatalf("accepted invalid confirmation %q", value)
		}
	}
	if !validateRestoreConfirmation("RESTORE DATABASE") {
		t.Fatal("rejected exact restore confirmation")
	}
}
