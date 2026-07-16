package service

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hl6-server/internal/config"
)

func TestValidateBackupArchiveRejectsTraversal(t *testing.T) {
	archivePath := writeMaintenanceZIP(t, map[string]string{
		"../outside":    "x",
		"manifest.json": "{}",
	})
	svc := NewDatabaseMaintenanceService(nil, nil, &config.Config{MaintenanceDataDir: t.TempDir()}, NewDatabaseMaintenanceGate())
	if _, err := svc.ValidateArchive(context.Background(), archivePath); !errors.Is(err, ErrUnsafeArchive) {
		t.Fatalf("got %v, want ErrUnsafeArchive", err)
	}
}

func TestRestoreArgsDoNotUseClientFilename(t *testing.T) {
	svc := NewDatabaseMaintenanceService(nil, nil, &config.Config{DatabaseURL: "postgres://hl6:secret@db.example.test:5432/hl6?sslmode=require"}, NewDatabaseMaintenanceGate())
	args, err := svc.RestoreArgs(filepath.Join(t.TempDir(), "database.dump"))
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range args {
		if arg == "../../evil.dump" {
			t.Fatalf("untrusted client filename leaked into pg_restore arguments: %#v", args)
		}
	}
}

func TestRunPGRestoreReportsNotStartedWhenBinaryUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	svc := NewDatabaseMaintenanceService(nil, nil, &config.Config{
		DatabaseURL: "postgres://hl6:secret@db.example.test:5432/hl6?sslmode=require",
	}, NewDatabaseMaintenanceGate())

	started, err := svc.runPGRestore(context.Background(), filepath.Join(t.TempDir(), "database.dump"))
	if err == nil {
		t.Fatal("missing pg_restore binary did not fail")
	}
	if started {
		t.Fatal("restore was reported as started when pg_restore could not start")
	}
}

func TestVerifyBackupArchiveRejectsRecordedChecksumMismatch(t *testing.T) {
	archivePath, checksum := writeVerifiedMaintenanceArchive(t)
	if err := VerifyBackupArchive(context.Background(), archivePath, checksum); err != nil {
		t.Fatalf("valid archive rejected: %v", err)
	}
	if err := VerifyBackupArchive(context.Background(), archivePath, strings.Repeat("0", 64)); !errors.Is(err, ErrInvalidBackupArchive) {
		t.Fatalf("got %v, want ErrInvalidBackupArchive", err)
	}
}

func TestRestoreGateRemainsClosedAfterDestructiveFailure(t *testing.T) {
	gate := NewDatabaseMaintenanceGate()
	if err := gate.BeginRestore(); err != nil {
		t.Fatal(err)
	}
	finishRestoreAttempt(gate, true)
	if !gate.IsRestoring() {
		t.Fatal("maintenance mode ended after a destructive restore attempt")
	}
}

func TestRestoreGateEndsBeforeDestructiveFailure(t *testing.T) {
	gate := NewDatabaseMaintenanceGate()
	if err := gate.BeginRestore(); err != nil {
		t.Fatal(err)
	}
	finishRestoreAttempt(gate, false)
	if gate.IsRestoring() {
		t.Fatal("maintenance mode remained after a non-destructive restore failure")
	}
}

func writeVerifiedMaintenanceArchive(t *testing.T) (string, string) {
	t.Helper()
	directory := t.TempDir()
	dumpPath := filepath.Join(directory, "database.dump")
	if err := os.WriteFile(dumpPath, []byte("PGDMPtest"), 0o600); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(directory, "archive.zip")
	svc := NewDatabaseMaintenanceService(nil, nil, nil, nil)
	checksum, err := svc.writeBackupArchive(dumpPath, archivePath, BackupManifest{SchemaVersion: backupSchemaVersion})
	if err != nil {
		t.Fatal(err)
	}
	return archivePath, checksum
}

func writeMaintenanceZIP(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, content := range entries {
		entry, createErr := writer.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(content)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
