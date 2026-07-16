package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
)

const (
	backupManifestEntry     = "manifest.json"
	backupChecksumEntry     = "SHA256SUMS"
	backupDumpEntry         = "database.dump"
	backupSchemaVersion     = 1
	maxBackupArchiveBytes   = int64(2 << 30)
	maxBackupExtractedBytes = int64(8 << 30)
	maxBackupArchiveFiles   = 3
	maxManifestBytes        = int64(1 << 20)
	backupRetention         = 7 * 24 * time.Hour
)

var (
	ErrUnsafeArchive        = errors.New("unsafe database backup archive")
	ErrInvalidBackupArchive = errors.New("invalid database backup archive")
	ErrRestoreInProgress    = errors.New("database restore is already in progress")
)

// MaintenanceGate is intentionally small so request middleware and restore
// orchestration share one in-process maintenance state without exposing the
// destructive implementation to HTTP handlers.
type MaintenanceGate interface {
	BeginRestore() error
	EndRestore()
	IsRestoring() bool
}

type DatabaseMaintenanceGate struct {
	restoring atomic.Bool
}

func NewDatabaseMaintenanceGate() *DatabaseMaintenanceGate {
	return &DatabaseMaintenanceGate{}
}

func (g *DatabaseMaintenanceGate) BeginRestore() error {
	if g == nil || !g.restoring.CompareAndSwap(false, true) {
		return ErrRestoreInProgress
	}
	return nil
}

func (g *DatabaseMaintenanceGate) EndRestore() {
	if g != nil {
		g.restoring.Store(false)
	}
}

func (g *DatabaseMaintenanceGate) IsRestoring() bool {
	return g != nil && g.restoring.Load()
}

type BackupManifest struct {
	SchemaVersion      int               `json:"schema_version"`
	CreatedAt          time.Time         `json:"created_at"`
	DatabaseVersion    string            `json:"database_version"`
	DatabaseIdentifier string            `json:"database_identifier"`
	Files              map[string]string `json:"files"`
}

type validatedBackupArchive struct {
	Manifest BackupManifest
	Checksum string
}

type DatabaseMaintenanceService struct {
	db   *gorm.DB
	repo *repository.Repository
	cfg  *config.Config
	gate MaintenanceGate
}

func NewDatabaseMaintenanceService(db *gorm.DB, repo *repository.Repository, cfg *config.Config, gate MaintenanceGate) *DatabaseMaintenanceService {
	if gate == nil {
		gate = NewDatabaseMaintenanceGate()
	}
	return &DatabaseMaintenanceService{db: db, repo: repo, cfg: cfg, gate: gate}
}

func (s *DatabaseMaintenanceService) Gate() MaintenanceGate {
	return s.gate
}

// CreateBackup performs a server-controlled pg_dump and packages it in the
// only archive format that Restore accepts. Callers cannot choose executable,
// database URL, SQL, path, or archive entry names.
func (s *DatabaseMaintenanceService) CreateBackup(ctx context.Context, userID uint) (*model.DatabaseBackup, error) {
	if s == nil || s.db == nil || s.repo == nil || s.cfg == nil || userID == 0 {
		return nil, errors.New("database maintenance service is unavailable")
	}
	directory, err := s.ensureStorageDirectory()
	if err != nil {
		return nil, err
	}
	dumpFile, err := os.CreateTemp(directory, ".hl6-export-*.dump")
	if err != nil {
		return nil, fmt.Errorf("create database export file: %w", err)
	}
	dumpPath := dumpFile.Name()
	if err := dumpFile.Close(); err != nil {
		_ = os.Remove(dumpPath)
		return nil, fmt.Errorf("close database export file: %w", err)
	}
	defer os.Remove(dumpPath)

	if err := s.runPGDump(ctx, dumpPath); err != nil {
		return nil, err
	}
	databaseVersion, err := s.postgreSQLVersion(ctx)
	if err != nil {
		return nil, err
	}
	archiveName := fmt.Sprintf("hl6-backup-%s-%s.zip", time.Now().UTC().Format("20060102T150405Z"), uuid.NewString())
	archivePath := filepath.Join(directory, archiveName)
	checksum, err := s.writeBackupArchive(dumpPath, archivePath, BackupManifest{
		SchemaVersion:      backupSchemaVersion,
		CreatedAt:          time.Now().UTC(),
		DatabaseVersion:    databaseVersion,
		DatabaseIdentifier: databaseIdentifier(s.cfg.DatabaseURL),
	})
	if err != nil {
		_ = os.Remove(archivePath)
		return nil, err
	}
	if _, err := s.ValidateArchive(ctx, archivePath); err != nil {
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("verify generated backup archive: %w", err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(backupRetention)
	backup := &model.DatabaseBackup{
		CreatedByUserID: userID,
		Filename:        archiveName,
		ChecksumSHA256:  checksum,
		DatabaseVersion: databaseVersion,
		SchemaRevision:  "v2",
		StoragePath:     archivePath,
		Status:          model.DatabaseBackupStatusReady,
		ExpiresAt:       &expiresAt,
	}
	if err := s.repo.CreateDatabaseBackup(backup); err != nil {
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("record database export: %w", err)
	}
	return backup, nil
}

// ValidateArchive validates both the ZIP container and HL6 manifest before an
// archive is ever extracted or passed to pg_restore.
func (s *DatabaseMaintenanceService) ValidateArchive(ctx context.Context, archivePath string) (*BackupManifest, error) {
	validated, err := validateBackupArchive(ctx, archivePath)
	if err != nil {
		return nil, err
	}
	return &validated.Manifest, nil
}

// VerifyBackupArchive confirms that a server-recorded archive is still a
// regular, valid HL6 backup and has not changed since its checksum was saved.
func VerifyBackupArchive(ctx context.Context, archivePath, expectedChecksum string) error {
	if len(expectedChecksum) != 64 {
		return ErrInvalidBackupArchive
	}
	validated, err := validateBackupArchive(ctx, archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(validated.Checksum, expectedChecksum) {
		return ErrInvalidBackupArchive
	}
	return nil
}

// StoreUploadedArchive copies the request stream to a server-created path
// while enforcing a compressed-size limit. The original multipart filename is
// intentionally ignored.
func (s *DatabaseMaintenanceService) StoreUploadedArchive(ctx context.Context, source io.Reader) (string, error) {
	if s == nil || source == nil {
		return "", ErrInvalidBackupArchive
	}
	directory, err := s.ensureStorageDirectory()
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(directory, ".hl6-upload-*.zip")
	if err != nil {
		return "", fmt.Errorf("create uploaded archive file: %w", err)
	}
	path := file.Name()
	limited := io.LimitReader(source, maxBackupArchiveBytes+1)
	written, copyErr := io.Copy(file, limited)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written > maxBackupArchiveBytes {
		_ = os.Remove(path)
		if written > maxBackupArchiveBytes {
			return "", ErrInvalidBackupArchive
		}
		if copyErr != nil {
			return "", fmt.Errorf("store uploaded archive: %w", copyErr)
		}
		return "", fmt.Errorf("close uploaded archive: %w", closeErr)
	}
	if err := ctx.Err(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

// Restore validates an upload, creates a mandatory pre-restore export, then
// runs fixed pg_restore arguments. On success the gate remains enabled until
// the process restarts, avoiding requests served with stale database state.
func (s *DatabaseMaintenanceService) Restore(ctx context.Context, userID uint, challengeHash, archivePath string) (*model.DatabaseRestoreJob, error) {
	if s == nil || s.db == nil || s.repo == nil || s.cfg == nil || userID == 0 || len(challengeHash) != 64 {
		return nil, errors.New("database restore service is unavailable")
	}
	inputChecksum, err := fileSHA256(archivePath)
	if err != nil {
		return nil, fmt.Errorf("checksum uploaded archive: %w", err)
	}
	job := &model.DatabaseRestoreJob{
		CreatedByUserID:     userID,
		ChallengeHash:       challengeHash,
		InputChecksumSHA256: inputChecksum,
		Status:              model.DatabaseRestoreStatusPending,
	}
	if err := s.repo.CreateDatabaseRestoreJob(job); err != nil {
		return nil, fmt.Errorf("create database restore job: %w", err)
	}

	if _, err := s.ValidateArchive(ctx, archivePath); err != nil {
		s.failRestoreJob(job, "archive validation failed")
		s.recordAudit(userID, "admin_database_restore_validation_failed", "database_restore", job.ID, map[string]string{"reason": "archive_validation_failed"})
		return job, err
	}
	preRestoreBackup, err := s.CreateBackup(ctx, userID)
	if err != nil {
		s.failRestoreJob(job, "mandatory pre-restore backup failed")
		s.recordAudit(userID, "admin_database_restore_failed", "database_restore", job.ID, map[string]string{"reason": "pre_restore_backup_failed"})
		return job, fmt.Errorf("create mandatory pre-restore backup: %w", err)
	}
	if err := s.gate.BeginRestore(); err != nil {
		s.failRestoreJob(job, "another restore is already in progress")
		return job, err
	}
	destructiveRestoreStarted := false
	defer func() {
		finishRestoreAttempt(s.gate, destructiveRestoreStarted)
	}()
	if err := s.repo.MarkDatabaseRestoreJobRunning(job, preRestoreBackup.ID); err != nil {
		return job, fmt.Errorf("mark database restore running: %w", err)
	}
	s.recordAudit(userID, "admin_database_restore_started", "database_restore", job.ID, map[string]uint{"pre_restore_backup_id": preRestoreBackup.ID})

	directory, err := s.ensureStorageDirectory()
	if err != nil {
		s.failRestoreJob(job, "restore storage is unavailable")
		return job, err
	}
	extractDirectory, err := os.MkdirTemp(directory, ".hl6-restore-")
	if err != nil {
		s.failRestoreJob(job, "create restore workspace failed")
		return job, fmt.Errorf("create restore workspace: %w", err)
	}
	defer os.RemoveAll(extractDirectory)
	if err := extractValidatedBackup(ctx, archivePath, extractDirectory); err != nil {
		s.failRestoreJob(job, "extract verified archive failed")
		return job, err
	}
	dumpPath := filepath.Join(extractDirectory, backupDumpEntry)
	destructiveRestoreStarted, err = s.runPGRestore(ctx, dumpPath)
	if err != nil {
		s.failRestoreJob(job, "pg_restore failed")
		s.recordAudit(userID, "admin_database_restore_failed", "database_restore", job.ID, map[string]string{"reason": "pg_restore_failed"})
		return job, err
	}
	if err := s.validateRestoredDatabase(ctx); err != nil {
		s.failRestoreJob(job, "post-restore validation failed")
		s.recordAudit(userID, "admin_database_restore_failed", "database_restore", job.ID, map[string]string{"reason": "validation_failed"})
		return job, err
	}
	completedJob, err := s.persistRestoredOutcome(job, preRestoreBackup)
	if err != nil {
		return job, err
	}
	s.recordAudit(userID, "admin_database_restore_succeeded", "database_restore", completedJob.ID, map[string]uint{"pre_restore_backup_id": *completedJob.PreRestoreBackupID})
	return completedJob, nil
}

// finishRestoreAttempt only reopens the API when no destructive database
// command has started. After pg_restore begins, a restart and operator review
// are required even when pg_restore returns an error.
func finishRestoreAttempt(gate MaintenanceGate, destructiveRestoreStarted bool) {
	if gate != nil && !destructiveRestoreStarted {
		gate.EndRestore()
	}
}

// persistRestoredOutcome recreates restore metadata after pg_restore replaces
// the tables that held the in-progress job and its mandatory safety backup.
func (s *DatabaseMaintenanceService) persistRestoredOutcome(previousJob *model.DatabaseRestoreJob, preRestoreBackup *model.DatabaseBackup) (*model.DatabaseRestoreJob, error) {
	if s == nil || s.repo == nil || previousJob == nil || preRestoreBackup == nil {
		return nil, errors.New("restore outcome persistence is unavailable")
	}
	recordedBackup := *preRestoreBackup
	recordedBackup.ID = 0
	if err := s.repo.CreateDatabaseBackup(&recordedBackup); err != nil {
		return nil, fmt.Errorf("record restored pre-restore backup: %w", err)
	}
	now := time.Now().UTC()
	completed := &model.DatabaseRestoreJob{
		CreatedByUserID:     previousJob.CreatedByUserID,
		ChallengeHash:       previousJob.ChallengeHash,
		InputChecksumSHA256: previousJob.InputChecksumSHA256,
		PreRestoreBackupID:  &recordedBackup.ID,
		Status:              model.DatabaseRestoreStatusSucceeded,
		ValidationResult:    "database health and required tables validated",
		StartedAt:           previousJob.StartedAt,
		FinishedAt:          &now,
	}
	if err := s.repo.CreateDatabaseRestoreJob(completed); err != nil {
		return nil, fmt.Errorf("record restored database restore job: %w", err)
	}
	return completed, nil
}

func (s *DatabaseMaintenanceService) RestoreArgs(dumpPath string) ([]string, error) {
	if s == nil || s.cfg == nil || !filepath.IsAbs(dumpPath) || filepath.Base(dumpPath) != backupDumpEntry {
		return nil, ErrInvalidBackupArchive
	}
	databaseURL, _, err := sanitizedPostgresConnection(s.cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return []string{
		"--clean",
		"--if-exists",
		"--no-owner",
		"--no-privileges",
		"--exit-on-error",
		"--dbname=" + databaseURL,
		dumpPath,
	}, nil
}

func (s *DatabaseMaintenanceService) ensureStorageDirectory() (string, error) {
	directory := ""
	if s.cfg != nil {
		directory = strings.TrimSpace(s.cfg.MaintenanceDataDir)
	}
	if directory == "" {
		return "", errors.New("maintenance data directory is not configured")
	}
	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absDirectory, 0o700); err != nil {
		return "", fmt.Errorf("create maintenance data directory: %w", err)
	}
	if err := os.Chmod(absDirectory, 0o700); err != nil {
		return "", fmt.Errorf("protect maintenance data directory: %w", err)
	}
	return absDirectory, nil
}

func (s *DatabaseMaintenanceService) runPGDump(ctx context.Context, dumpPath string) error {
	databaseURL, environment, err := sanitizedPostgresConnection(s.cfg.DatabaseURL)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "pg_dump",
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"--file="+dumpPath,
		"--dbname="+databaseURL,
	)
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %s", commandFailureSummary(output))
	}
	return nil
}

// runPGRestore reports whether the external process actually started. A
// failed executable lookup is recoverable, while a started pg_restore keeps
// maintenance mode enabled for operator review even if the command fails.
func (s *DatabaseMaintenanceService) runPGRestore(ctx context.Context, dumpPath string) (bool, error) {
	args, err := s.RestoreArgs(dumpPath)
	if err != nil {
		return false, err
	}
	_, environment, err := sanitizedPostgresConnection(s.cfg.DatabaseURL)
	if err != nil {
		return false, err
	}
	command := exec.CommandContext(ctx, "pg_restore", args...)
	command.Env = environment
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		return false, fmt.Errorf("start pg_restore: %w", err)
	}
	err = command.Wait()
	if err != nil {
		return true, fmt.Errorf("pg_restore failed: %s", commandFailureSummary(output.Bytes()))
	}
	return true, nil
}

func (s *DatabaseMaintenanceService) postgreSQLVersion(ctx context.Context) (string, error) {
	var version string
	if err := s.db.WithContext(ctx).Raw("SELECT current_setting('server_version')").Scan(&version).Error; err != nil {
		return "", fmt.Errorf("read PostgreSQL version: %w", err)
	}
	return strings.TrimSpace(version), nil
}

func (s *DatabaseMaintenanceService) writeBackupArchive(dumpPath, archivePath string, manifest BackupManifest) (string, error) {
	dumpChecksum, err := fileSHA256(dumpPath)
	if err != nil {
		return "", err
	}
	manifest.Files = map[string]string{backupDumpEntry: dumpChecksum}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestChecksum := sha256Hex(manifestBytes)
	checksums := []byte(fmt.Sprintf("%s  %s\n%s  %s\n", dumpChecksum, backupDumpEntry, manifestChecksum, backupManifestEntry))

	archiveFile, err := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create backup archive: %w", err)
	}
	writer := zip.NewWriter(archiveFile)
	writeErr := writeArchiveFile(writer, backupDumpEntry, dumpPath)
	if writeErr == nil {
		writeErr = writeArchiveBytes(writer, backupManifestEntry, manifestBytes)
	}
	if writeErr == nil {
		writeErr = writeArchiveBytes(writer, backupChecksumEntry, checksums)
	}
	if closeErr := writer.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if closeErr := archiveFile.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		return "", fmt.Errorf("write backup archive: %w", writeErr)
	}
	return fileSHA256(archivePath)
}

func writeArchiveFile(writer *zip.Writer, entryName, sourcePath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	entry, err := writer.Create(entryName)
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, source)
	return err
}

func writeArchiveBytes(writer *zip.Writer, entryName string, value []byte) error {
	entry, err := writer.Create(entryName)
	if err != nil {
		return err
	}
	_, err = entry.Write(value)
	return err
}

func validateBackupArchive(ctx context.Context, archivePath string) (validatedBackupArchive, error) {
	info, err := os.Stat(archivePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxBackupArchiveBytes {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	defer reader.Close()
	files := make(map[string]*zip.File, len(reader.File))
	var extractedSize int64
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return validatedBackupArchive{}, err
		}
		if !isExpectedArchiveEntry(file.Name) || !isSafeArchiveFile(file) {
			return validatedBackupArchive{}, ErrUnsafeArchive
		}
		if _, exists := files[file.Name]; exists {
			return validatedBackupArchive{}, ErrInvalidBackupArchive
		}
		extractedSize += int64(file.UncompressedSize64)
		if extractedSize > maxBackupExtractedBytes {
			return validatedBackupArchive{}, ErrInvalidBackupArchive
		}
		files[file.Name] = file
	}
	if len(files) != maxBackupArchiveFiles {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	for _, expected := range []string{backupDumpEntry, backupManifestEntry, backupChecksumEntry} {
		if files[expected] == nil {
			return validatedBackupArchive{}, ErrInvalidBackupArchive
		}
	}

	manifestBytes, err := readZipFile(files[backupManifestEntry], maxManifestBytes)
	if err != nil {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	var manifest BackupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil || manifest.SchemaVersion != backupSchemaVersion || manifest.Files[backupDumpEntry] == "" {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	checksumBytes, err := readZipFile(files[backupChecksumEntry], maxManifestBytes)
	if err != nil {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	sums, err := parseArchiveChecksums(string(checksumBytes))
	if err != nil {
		return validatedBackupArchive{}, err
	}
	if sums[backupManifestEntry] != sha256Hex(manifestBytes) || sums[backupDumpEntry] != manifest.Files[backupDumpEntry] {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	dumpChecksum, err := hashZipFile(files[backupDumpEntry])
	if err != nil || dumpChecksum != sums[backupDumpEntry] {
		return validatedBackupArchive{}, ErrInvalidBackupArchive
	}
	if err := ensureCustomDump(files[backupDumpEntry]); err != nil {
		return validatedBackupArchive{}, err
	}
	archiveChecksum, err := fileSHA256(archivePath)
	if err != nil {
		return validatedBackupArchive{}, err
	}
	return validatedBackupArchive{Manifest: manifest, Checksum: archiveChecksum}, nil
}

func extractValidatedBackup(ctx context.Context, archivePath, destination string) error {
	if _, err := validateBackupArchive(ctx, archivePath); err != nil {
		return err
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return ErrInvalidBackupArchive
	}
	defer reader.Close()
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		outputPath := filepath.Join(destination, file.Name)
		if filepath.Dir(outputPath) != destination {
			return ErrUnsafeArchive
		}
		input, err := file.Open()
		if err != nil {
			return ErrInvalidBackupArchive
		}
		output, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, io.LimitReader(input, int64(file.UncompressedSize64)+1))
		closeOutputErr := output.Close()
		closeInputErr := input.Close()
		if copyErr != nil || closeOutputErr != nil || closeInputErr != nil {
			return ErrInvalidBackupArchive
		}
	}
	return nil
}

func isExpectedArchiveEntry(name string) bool {
	return name == backupDumpEntry || name == backupManifestEntry || name == backupChecksumEntry
}

func isSafeArchiveFile(file *zip.File) bool {
	if file == nil || file.FileInfo().IsDir() || file.Mode()&os.ModeType != 0 {
		return false
	}
	if file.Name == "" || strings.HasPrefix(file.Name, "/") || strings.Contains(file.Name, "\\") || path.Clean(file.Name) != file.Name || strings.Contains(file.Name, "..") {
		return false
	}
	return file.UncompressedSize64 <= uint64(maxBackupExtractedBytes) && file.CompressedSize64 <= uint64(maxBackupArchiveBytes)
}

func readZipFile(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	value, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil || int64(len(value)) > limit {
		return nil, ErrInvalidBackupArchive
	}
	return value, nil
}

func hashZipFile(file *zip.File) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ensureCustomDump(file *zip.File) error {
	reader, err := file.Open()
	if err != nil {
		return ErrInvalidBackupArchive
	}
	defer reader.Close()
	magic := make([]byte, 5)
	if _, err := io.ReadFull(reader, magic); err != nil || string(magic) != "PGDMP" {
		return ErrInvalidBackupArchive
	}
	return nil
}

func parseArchiveChecksums(raw string) (map[string]string, error) {
	result := make(map[string]string, 2)
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 || len(parts[0]) != 64 || !isExpectedArchiveEntry(parts[1]) || parts[1] == backupChecksumEntry {
			return nil, ErrInvalidBackupArchive
		}
		if _, err := hex.DecodeString(parts[0]); err != nil {
			return nil, ErrInvalidBackupArchive
		}
		if _, exists := result[parts[1]]; exists {
			return nil, ErrInvalidBackupArchive
		}
		result[parts[1]] = strings.ToLower(parts[0])
	}
	if len(result) != 2 || result[backupDumpEntry] == "" || result[backupManifestEntry] == "" {
		return nil, ErrInvalidBackupArchive
	}
	return result, nil
}

func sanitizedPostgresConnection(raw string) (string, []string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" || strings.Trim(parsed.Path, "/") == "" {
		return "", nil, errors.New("invalid configured PostgreSQL connection")
	}
	username := ""
	password := ""
	if parsed.User != nil {
		username = parsed.User.Username()
		password, _ = parsed.User.Password()
	}
	if username == "" {
		return "", nil, errors.New("configured PostgreSQL username is missing")
	}
	parsed.User = url.User(username)
	environment := append([]string{}, os.Environ()...)
	if password != "" {
		environment = append(environment, "PGPASSWORD="+password)
	}
	return parsed.String(), environment, nil
}

func databaseIdentifier(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.Trim(parsed.Host+parsed.EscapedPath(), "/")
}

func fileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sha256Hex(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

func commandFailureSummary(output []byte) string {
	summary := strings.TrimSpace(string(output))
	if summary == "" {
		return "command returned a non-zero status"
	}
	if len(summary) > 500 {
		return summary[:500]
	}
	return summary
}

func (s *DatabaseMaintenanceService) validateRestoredDatabase(ctx context.Context) error {
	if err := s.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	for _, table := range []interface{}{&model.User{}, &model.UserCredential{}, &model.SystemConfig{}} {
		if !s.db.Migrator().HasTable(table) {
			return fmt.Errorf("restored database is missing required table %T", table)
		}
	}
	return nil
}

func (s *DatabaseMaintenanceService) failRestoreJob(job *model.DatabaseRestoreJob, detail string) {
	if job == nil || s == nil || s.repo == nil {
		return
	}
	_ = s.repo.FinishDatabaseRestoreJob(job, model.DatabaseRestoreStatusFailed, "", detail)
}

func (s *DatabaseMaintenanceService) recordAudit(userID uint, action, resource string, resourceID uint, details interface{}) {
	if s == nil || s.repo == nil || userID == 0 {
		return
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return
	}
	_ = s.repo.CreateAuditLog(&model.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    encoded,
	})
}
