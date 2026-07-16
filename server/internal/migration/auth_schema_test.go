package migration

import (
	"testing"

	"hl6-server/internal/model"
)

func TestEnsureLocalAuthDefaultEnablesOnlyEmptyInstallations(t *testing.T) {
	db := newAuthCutoverTestDB(t)

	if err := EnsureLocalAuthDefault(db); err != nil {
		t.Fatal(err)
	}
	var localAuth model.SystemConfig
	if err := db.Where("\"key\" = ?", "auth.local.enabled").First(&localAuth).Error; err != nil {
		t.Fatal(err)
	}
	if localAuth.Value != "true" {
		t.Fatalf("empty installation local auth = %q, want true", localAuth.Value)
	}

	if err := db.Where("\"key\" = ?", "auth.local.enabled").Delete(&model.SystemConfig{}).Error; err != nil {
		t.Fatal(err)
	}
	group := model.UserGroup{Name: "Default", IsDefault: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.User{Email: "legacy@example.com", Name: "Legacy", Role: "user", GroupID: &group.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureLocalAuthDefault(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("\"key\" = ?", "auth.local.enabled").First(&localAuth).Error; err != nil {
		t.Fatal(err)
	}
	if localAuth.Value != "false" {
		t.Fatalf("legacy installation local auth = %q, want false", localAuth.Value)
	}
}
