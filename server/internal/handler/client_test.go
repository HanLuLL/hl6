package handler

import "testing"

func TestClientRecoveryConfigForcesUpdateAfterKeyRevocation(t *testing.T) {
	config := clientRecoveryConfigResponse(map[string]string{
		clientLatestVersionConfigKey: "1.0.1",
		clientForceUpdateConfigKey:   "false",
		clientUpdateNoticeConfigKey:  "Install the replacement client.",
		clientUpdateURLConfigKey:     "https://downloads.example.test/hl6.apk",
	})
	if config["force_update"] != true || config["update_available"] != true || config["communication_key_invalid"] != true {
		t.Fatalf("unexpected recovery config: %#v", config)
	}
}
