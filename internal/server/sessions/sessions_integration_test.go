package sessions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/servertest"
	"github.com/ilcm96/codex-usage/internal/server/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRepository_PublicMethods(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	fixture := servertest.SeedFixture(t, ctx, db)

	repository := sessions.NewPostgresRepository(db)

	t.Run("ListSimple", func(t *testing.T) {
		got, err := repository.ListSimple(ctx, sessions.ListParams{Limit: 10, Sort: "updated_desc", Query: "cache"})
		if err != nil {
			t.Fatalf("ListSimple failed: %v", err)
		}
		if len(got) != 2 || got[0].ID != fixture.SessionAlpha || got[0].TotalTokens != 300 {
			t.Fatalf("unexpected ListSimple result: %+v", got)
		}
	})

	t.Run("List", func(t *testing.T) {
		got, err := repository.List(ctx, sessions.ListParams{Limit: 1, Offset: 0, Sort: "cost_desc", Model: "gpt-5-mini"})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if got.Total != 1 || got.NextOffset != 0 || len(got.Items) != 1 || got.Items[0].ID != fixture.SessionAlpha {
			t.Fatalf("unexpected List result: %+v", got)
		}
		if got.Totals.TotalTokens != 300 || got.Totals.ToolCalls != 1 {
			t.Fatalf("unexpected List totals: %+v", got.Totals)
		}
	})

	t.Run("List paginates and filters by ids date and branch", func(t *testing.T) {
		today := currentDate(t, ctx, db, "CURRENT_DATE")
		got, err := repository.List(ctx, sessions.ListParams{
			Limit:        1,
			Offset:       0,
			Sort:         "tokens_asc",
			From:         today,
			To:           today,
			DeviceID:     fixture.DeviceID,
			RepositoryID: fixture.RepositoryID,
			ProjectID:    fixture.ProjectID,
			Branch:       "main",
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if got.Total != 1 || got.NextOffset != 0 || len(got.Items) != 1 || got.Items[0].ID != fixture.SessionAlpha {
			t.Fatalf("unexpected filtered List result: %+v", got)
		}

		paged, err := repository.List(ctx, sessions.ListParams{Limit: 1, Offset: 0, Sort: "tokens_asc"})
		if err != nil {
			t.Fatalf("List pagination failed: %v", err)
		}
		if paged.Total != 2 || paged.NextOffset != 1 || len(paged.Items) != 1 || paged.Items[0].ID != fixture.SessionBeta {
			t.Fatalf("unexpected paged List result: %+v", paged)
		}
	})

	t.Run("List returns empty result for invalid id filter", func(t *testing.T) {
		got, err := repository.List(ctx, sessions.ListParams{Limit: 10, Sort: "updated_desc", DeviceID: "not-a-uuid"})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if got.Total != 0 || len(got.Items) != 0 {
			t.Fatalf("unexpected invalid id List result: %+v", got)
		}
	})

	t.Run("ListItems", func(t *testing.T) {
		got, err := repository.ListItems(ctx, sessions.ListParams{Limit: 2, Offset: 0, Sort: "tokens_asc"})
		if err != nil {
			t.Fatalf("ListItems failed: %v", err)
		}
		if len(got) != 2 || got[0].ID != fixture.SessionBeta || got[1].ID != fixture.SessionAlpha {
			t.Fatalf("unexpected ListItems order: %+v", got)
		}
	})

	t.Run("Detail", func(t *testing.T) {
		got, err := repository.Detail(ctx, fixture.SessionAlpha)
		if err != nil {
			t.Fatalf("Detail failed: %v", err)
		}
		if got.Session.ID != fixture.SessionAlpha || got.Session.DisplayTitle != "Cache tuning" || len(got.Models) != 2 || got.Models[0].Model != "gpt-5" {
			t.Fatalf("unexpected Detail result: %+v", got)
		}
		_, err = repository.Detail(ctx, "missing-session")
		if !errors.Is(err, sessions.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Reader", func(t *testing.T) {
		got, err := repository.Reader(ctx, fixture.SessionAlpha, sessions.ReaderParams{Limit: 10, Offset: 0, Query: "cache"})
		if err != nil {
			t.Fatalf("Reader failed: %v", err)
		}
		if got.Total != 1 || got.Summary.SessionID != fixture.SessionAlpha || got.Items[0].ToolCallCount != 1 {
			t.Fatalf("unexpected Reader result: %+v", got)
		}
		_, err = repository.Reader(ctx, "missing-session", sessions.ReaderParams{Limit: 10})
		if !errors.Is(err, sessions.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Timeline", func(t *testing.T) {
		got, err := repository.Timeline(ctx, fixture.SessionAlpha, sessions.TimelineParams{Limit: 10, Offset: 0, Kind: "tool"})
		if err != nil {
			t.Fatalf("Timeline failed: %v", err)
		}
		if got.Total != 1 || len(got.Items) != 1 || got.Items[0].ToolName != "shell" {
			t.Fatalf("unexpected Timeline result: %+v", got)
		}
	})

	t.Run("Timeline query filters message text", func(t *testing.T) {
		got, err := repository.Timeline(ctx, fixture.SessionAlpha, sessions.TimelineParams{Limit: 10, Offset: 0, Kind: "message", Query: "Testcontainers"})
		if err != nil {
			t.Fatalf("Timeline failed: %v", err)
		}
		if got.Total != 1 || len(got.Items) != 1 || got.Items[0].Role != "assistant" {
			t.Fatalf("unexpected Timeline query result: %+v", got)
		}
	})

	t.Run("RawPath", func(t *testing.T) {
		got, err := repository.RawPath(ctx, fixture.SessionAlpha)
		if err != nil {
			t.Fatalf("RawPath failed: %v", err)
		}
		if got != fixture.RawPath {
			t.Fatalf("unexpected RawPath: got %q want %q", got, fixture.RawPath)
		}
		_, err = repository.RawPath(ctx, "missing-session")
		if !errors.Is(err, sessions.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("FullTimeline", func(t *testing.T) {
		got, err := repository.FullTimeline(ctx, fixture.SessionAlpha)
		if err != nil {
			t.Fatalf("FullTimeline failed: %v", err)
		}
		if len(got) != 3 || got[0].Kind != "message" || got[1].Kind != "tool" || got[2].Role != "assistant" {
			t.Fatalf("unexpected FullTimeline result: %+v", got)
		}
	})
}

func TestService_PublicMethodsWithPostgresRepository(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	fixture := servertest.SeedFixture(t, ctx, db)

	service := sessions.NewService(sessions.NewPostgresRepository(db))

	if got, err := service.ListSimple(ctx, sessions.ListParams{Limit: 999, Sort: "updated_desc"}); err != nil || len(got) != 2 {
		t.Fatalf("ListSimple failed: got=%+v err=%v", got, err)
	}
	if _, err := service.ListSimple(ctx, sessions.ListParams{Sort: "unknown"}); !errors.Is(err, sessions.ErrUnsupportedSort) {
		t.Fatalf("expected ErrUnsupportedSort, got %v", err)
	}

	list, err := service.List(ctx, sessions.ListParams{Limit: 999, Offset: -10, Sort: "updated_desc"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if list.Limit != 200 || list.Offset != 0 || list.Total != 2 {
		t.Fatalf("unexpected List normalization: %+v", list)
	}
	if _, err := service.List(ctx, sessions.ListParams{Sort: "unknown"}); !errors.Is(err, sessions.ErrUnsupportedSort) {
		t.Fatalf("expected ErrUnsupportedSort, got %v", err)
	}

	items, err := service.ListItems(ctx, sessions.ListParams{Limit: 999, Offset: -10, Sort: "updated_desc"})
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected ListItems result: %+v", items)
	}
	if _, err := service.ListItems(ctx, sessions.ListParams{Sort: "unknown"}); !errors.Is(err, sessions.ErrUnsupportedSort) {
		t.Fatalf("expected ErrUnsupportedSort, got %v", err)
	}

	detail, err := service.Detail(ctx, fixture.SessionAlpha)
	if err != nil || detail.Session.ID != fixture.SessionAlpha {
		t.Fatalf("Detail failed: got=%+v err=%v", detail, err)
	}

	reader, err := service.Reader(ctx, fixture.SessionAlpha, sessions.ReaderParams{Limit: 999, Offset: -10})
	if err != nil {
		t.Fatalf("Reader failed: %v", err)
	}
	if reader.Limit != 100 || reader.Offset != 0 || reader.Total != 1 {
		t.Fatalf("unexpected Reader normalization: %+v", reader)
	}

	timeline, err := service.Timeline(ctx, fixture.SessionAlpha, sessions.TimelineParams{Limit: 999, Offset: -10, Kind: "message"})
	if err != nil {
		t.Fatalf("Timeline failed: %v", err)
	}
	if timeline.Limit != 300 || timeline.Offset != 0 || timeline.Total != 2 {
		t.Fatalf("unexpected Timeline normalization: %+v", timeline)
	}
	if _, err := service.Timeline(ctx, fixture.SessionAlpha, sessions.TimelineParams{Kind: "unknown"}); !errors.Is(err, sessions.ErrUnsupportedKind) {
		t.Fatalf("expected ErrUnsupportedKind, got %v", err)
	}
}

func currentDate(t *testing.T, ctx context.Context, db *pgxpool.Pool, expression string) string {
	t.Helper()

	var out string
	if err := db.QueryRow(ctx, "SELECT ("+expression+")::text").Scan(&out); err != nil {
		t.Fatalf("load current date: %v", err)
	}
	return out
}
