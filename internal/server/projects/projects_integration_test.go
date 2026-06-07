package projects_test

import (
	"context"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/projects"
	"github.com/ilcm96/codex-usage/internal/server/servertest"
)

func TestPostgresRepository_PublicMethods(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	servertest.SeedFixture(t, ctx, db)

	repository := projects.NewPostgresRepository(db)

	repositories, err := repository.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("ListRepositories failed: %v", err)
	}
	if len(repositories) != 1 || repositories[0].Name != "codex-usage" || repositories[0].Sessions != 2 || repositories[0].TotalTokens != 380 {
		t.Fatalf("unexpected ListRepositories result: %+v", repositories)
	}

	projectList, err := repository.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projectList) != 1 || projectList[0].DisplayName != "codex-usage" || projectList[0].Sessions != 2 || projectList[0].CostUSD != 0.04 {
		t.Fatalf("unexpected ListProjects result: %+v", projectList)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO repositories (id, repository_url, repository_host, repository_owner, repository_name)
		VALUES ('00000000-0000-0000-0000-000000000102', 'https://github.com/acme/empty', 'github.com', 'acme', 'empty')
	`)
	if err != nil {
		t.Fatalf("insert empty repository: %v", err)
	}
	_, err = db.Exec(ctx, `
		INSERT INTO projects (id, repository_id, cwd, git_root, relative_path, display_name)
		VALUES (
			'00000000-0000-0000-0000-000000000202',
			'00000000-0000-0000-0000-000000000102',
			'/repo/empty', '/repo/empty', '.', 'empty'
		)
	`)
	if err != nil {
		t.Fatalf("insert empty project: %v", err)
	}

	repositories, err = repository.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("ListRepositories with empty repository failed: %v", err)
	}
	if len(repositories) != 2 || repositories[1].Name != "empty" || repositories[1].Sessions != 0 || repositories[1].TotalTokens != 0 {
		t.Fatalf("unexpected empty repository summary: %+v", repositories)
	}

	projectList, err = repository.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects with empty project failed: %v", err)
	}
	if len(projectList) != 2 || projectList[1].DisplayName != "empty" || projectList[1].Sessions != 0 || projectList[1].CostUSD != 0 {
		t.Fatalf("unexpected empty project summary: %+v", projectList)
	}
}

func TestService_PublicMethodsWithPostgresRepository(t *testing.T) {
	ctx := context.Background()
	db := servertest.StartPostgres(t)
	servertest.Reset(t, ctx, db)
	servertest.SeedFixture(t, ctx, db)

	service := projects.NewService(projects.NewPostgresRepository(db))

	repositories, err := service.ListRepositories(ctx)
	if err != nil || len(repositories) != 1 {
		t.Fatalf("ListRepositories failed: got=%+v err=%v", repositories, err)
	}

	projectList, err := service.ListProjects(ctx)
	if err != nil || len(projectList) != 1 {
		t.Fatalf("ListProjects failed: got=%+v err=%v", projectList, err)
	}
}
