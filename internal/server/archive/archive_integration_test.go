package archive_test

import (
	"context"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/archive"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestPostgresRepository_PublicMethods(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	fixture := servertest.SeedFixture(t, ctx, db)

	repository := archive.NewPostgresRepository(db)

	status, err := repository.Status(ctx)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.Sessions != 2 || status.Devices != 1 || status.Messages != 3 || status.ToolEvents != 1 || status.UsageEvents != 3 || status.MissingRawFiles != 1 || status.MissingRawSHA != 1 {
		t.Fatalf("unexpected Status result: %+v", status)
	}

	health, err := repository.Health(ctx)
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if health.Status != "attention" || health.Sessions != 2 || health.ArchiveRows != 1 || health.MissingArchiveRows != 1 || health.VerifiedArchiveRows != 1 {
		t.Fatalf("unexpected Health result: %+v", health)
	}

	byDevice, err := repository.ByDevice(ctx)
	if err != nil {
		t.Fatalf("ByDevice failed: %v", err)
	}
	if len(byDevice) != 1 || byDevice[0].Name != "Work MacBook" || byDevice[0].Sessions != 2 {
		t.Fatalf("unexpected ByDevice result: %+v", byDevice)
	}

	byRepository, err := repository.ByRepository(ctx, 1)
	if err != nil {
		t.Fatalf("ByRepository failed: %v", err)
	}
	if len(byRepository) != 1 || byRepository[0].Name != "codex-usage" || byRepository[0].Sessions != 2 {
		t.Fatalf("unexpected ByRepository result: %+v", byRepository)
	}

	integrity, err := repository.Integrity(ctx)
	if err != nil {
		t.Fatalf("Integrity failed: %v", err)
	}
	if integrity.Checked != 2 || integrity.OK != 1 || integrity.MissingSHA != 1 || len(integrity.Issues) != 1 {
		t.Fatalf("unexpected Integrity result: %+v", integrity)
	}

	_, err = db.Exec(ctx, `
		UPDATE sessions
		SET raw_file_path = $1, raw_sha256 = 'raw-beta-sha'
		WHERE id = $2
	`, fixture.RawPath, fixture.SessionBeta)
	if err != nil {
		t.Fatalf("update beta archive metadata: %v", err)
	}
	_, err = db.Exec(ctx, `
		INSERT INTO archive_files (
			session_id, device_id, raw_file_path, raw_sha256,
			raw_size_bytes, verified_at, status
		)
		VALUES ($1, $2, $3, 'raw-beta-sha', 80, now(), 'verified')
	`, fixture.SessionBeta, fixture.DeviceID, fixture.RawPath)
	if err != nil {
		t.Fatalf("insert beta archive row: %v", err)
	}

	okHealth, err := repository.Health(ctx)
	if err != nil {
		t.Fatalf("Health after archive completion failed: %v", err)
	}
	if okHealth.Status != "ok" || okHealth.MissingRawFiles != 0 || okHealth.MissingArchiveRows != 0 || okHealth.VerifiedArchiveRows != okHealth.ArchiveRows {
		t.Fatalf("unexpected ok Health result: %+v", okHealth)
	}
}

func TestService_PublicMethodsWithPostgresRepository(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	servertest.SeedFixture(t, ctx, db)

	service := archive.NewService(archive.NewPostgresRepository(db))

	if status, err := service.Status(ctx); err != nil || status.Sessions != 2 {
		t.Fatalf("Status failed: got=%+v err=%v", status, err)
	}
	if health, err := service.Health(ctx); err != nil || health.Status != "attention" {
		t.Fatalf("Health failed: got=%+v err=%v", health, err)
	}
	if byDevice, err := service.ByDevice(ctx); err != nil || len(byDevice) != 1 {
		t.Fatalf("ByDevice failed: got=%+v err=%v", byDevice, err)
	}
	if byRepository, err := service.ByRepository(ctx, 999); err != nil || len(byRepository) != 1 {
		t.Fatalf("ByRepository failed: got=%+v err=%v", byRepository, err)
	}
	if integrity, err := service.Integrity(ctx); err != nil || integrity.Checked != 2 {
		t.Fatalf("Integrity failed: got=%+v err=%v", integrity, err)
	}
}
