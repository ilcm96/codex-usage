package search_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/search"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRepository_Search(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	fixture := servertest.SeedFixture(t, ctx, db)

	repository := search.NewPostgresRepository(db)

	t.Run("message search with total", func(t *testing.T) {
		got, err := repository.Search(ctx, search.Params{Query: "cache", Limit: 10, IncludeTotal: true, Kind: "message"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if !got.TotalKnown || got.Total != 1 || len(got.Items) != 1 || got.Items[0].SessionID != fixture.SessionAlpha || got.Items[0].MatchStart < 0 {
			t.Fatalf("unexpected Search result: %+v", got)
		}
	})

	t.Run("tool search without total uses next offset", func(t *testing.T) {
		got, err := repository.Search(ctx, search.Params{Query: "postgres", Limit: 1, IncludeTotal: false, Kind: "all"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if got.TotalKnown || got.Total != 1 || got.NextOffset != 0 || len(got.Items) != 1 || got.Items[0].ToolName != "shell" {
			t.Fatalf("unexpected Search result: %+v", got)
		}
	})

	t.Run("filters by model and ids", func(t *testing.T) {
		got, err := repository.Search(ctx, search.Params{
			Query:        "cache",
			Limit:        10,
			IncludeTotal: true,
			Kind:         "user",
			Model:        "gpt-5-mini",
			DeviceID:     fixture.DeviceID,
			RepositoryID: fixture.RepositoryID,
			ProjectID:    fixture.ProjectID,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if got.Total != 1 || len(got.Items) != 1 || got.Items[0].DocumentScope != "user" {
			t.Fatalf("unexpected filtered Search result: %+v", got)
		}
	})

	t.Run("returns empty result for invalid id filter", func(t *testing.T) {
		got, err := repository.Search(ctx, search.Params{Query: "cache", Limit: 10, IncludeTotal: true, Kind: "message", DeviceID: "not-a-uuid"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if got.Total != 0 || len(got.Items) != 0 {
			t.Fatalf("unexpected invalid id Search result: %+v", got)
		}
	})

	t.Run("filters by date range", func(t *testing.T) {
		today := currentDate(t, ctx, db, "CURRENT_DATE")
		yesterday := currentDate(t, ctx, db, "CURRENT_DATE - 1")

		todayResult, err := repository.Search(ctx, search.Params{Query: "repository", Limit: 10, IncludeTotal: true, Kind: "message", From: today})
		if err != nil {
			t.Fatalf("Search today failed: %v", err)
		}
		if todayResult.Total != 0 || len(todayResult.Items) != 0 {
			t.Fatalf("unexpected today Search result: %+v", todayResult)
		}

		yesterdayResult, err := repository.Search(ctx, search.Params{Query: "repository", Limit: 10, IncludeTotal: true, Kind: "message", To: yesterday})
		if err != nil {
			t.Fatalf("Search yesterday failed: %v", err)
		}
		if yesterdayResult.Total != 1 || len(yesterdayResult.Items) != 1 || yesterdayResult.Items[0].SessionID != fixture.SessionBeta {
			t.Fatalf("unexpected yesterday Search result: %+v", yesterdayResult)
		}
	})

	t.Run("without total fetches one extra row for next offset", func(t *testing.T) {
		_, err := db.Exec(ctx, `
			INSERT INTO search_documents (
				session_id, seq, turn_index, occurred_at, kind, document_scope, tool_name, title, body,
				snippet, rank_weight, default_searchable
			)
			VALUES (
				'session-alpha', 4, 0, CURRENT_DATE + TIME '10:04', 'tool', 'tool', 'shell',
				'Postgres logs', 'postgres emitted another searchable line', '', 7, true
			)
		`)
		if err != nil {
			t.Fatalf("insert extra search document: %v", err)
		}

		got, err := repository.Search(ctx, search.Params{Query: "postgres", Limit: 1, IncludeTotal: false, Kind: "tool"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if got.TotalKnown || got.Total != 2 || got.NextOffset != 1 || len(got.Items) != 1 {
			t.Fatalf("unexpected paged Search result: %+v", got)
		}
	})
}

func currentDate(t *testing.T, ctx context.Context, db *pgxpool.Pool, expression string) string {
	t.Helper()

	var out string
	if err := db.QueryRow(ctx, "SELECT ("+expression+")::text").Scan(&out); err != nil {
		t.Fatalf("load current date: %v", err)
	}
	return out
}

func TestService_SearchWithPostgresRepository(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	servertest.SeedFixture(t, ctx, db)

	service := search.NewService(search.NewPostgresRepository(db))

	empty, err := service.Search(ctx, search.Params{Query: "   "})
	if err != nil {
		t.Fatalf("empty Search failed: %v", err)
	}
	if empty.Total != 0 || !empty.TotalKnown || len(empty.Items) != 0 {
		t.Fatalf("unexpected empty Search result: %+v", empty)
	}

	got, err := service.Search(ctx, search.Params{Query: "postgres", Limit: 999, Offset: -5, Kind: "tool"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if got.Limit != 200 || got.Offset != 0 || got.Total != 1 || got.Items[0].Kind != "tool" {
		t.Fatalf("unexpected Search normalization/result: %+v", got)
	}

	if _, err := service.Search(ctx, search.Params{Query: "postgres", Kind: "invalid"}); !errors.Is(err, search.ErrUnsupportedKind) {
		t.Fatalf("expected ErrUnsupportedKind, got %v", err)
	}
}
