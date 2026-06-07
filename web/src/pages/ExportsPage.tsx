import { BarChart3, Bot, CircleDollarSign, Download } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";

import { BulkExportLink } from "@/components/common/data-display";
import {
  EmptyState,
  MetricCard,
  PageTitle,
  Panel,
  PanelError,
  PanelLoading,
} from "@/components/common/primitives";
import { FacetSelect } from "@/components/filters/DashboardFilterBar";
import { ExportLinks } from "@/components/sessions/session-cards";
import { useApiData } from "@/lib/api";
import { fieldClassName } from "@/lib/constants";
import {
  formatMoney,
  formatNumber,
  sessionDisplaySummary,
  sessionDisplayTitle,
} from "@/lib/format";
import { buildSessionQuery, filtersFromSearchParams } from "@/lib/queries";
import type { DashboardFilters, Facets, SessionsListResponse } from "@/types";

export function ExportsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const facets = useApiData<Facets>("/api/filter-options");
  const filters = filtersFromSearchParams(searchParams);
  const query = searchParams.get("q") ?? "";
  const sort = searchParams.get("sort") ?? "updated_desc";
  const exportQuery = buildSessionQuery(filters, query, sort, 1000, 0);
  const sessions = useApiData<SessionsListResponse>(
    `/api/sessions?${buildSessionQuery(filters, query, sort, 20, 0)}`,
  );

  function updateFilter(key: keyof DashboardFilters, value: string) {
    const next = new URLSearchParams(searchParams);
    if (value) {
      next.set(key, value);
    } else {
      next.delete(key);
    }
    setSearchParams(next);
  }

  return (
    <div className="space-y-6">
      <PageTitle
        eyebrow={"Exports"}
        title={"Downloads"}
        description={"Download raw JSONL, normalized JSON, and Markdown files per session."}
      />

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          detail={`${sessions.data?.items.length ?? 0} sample rows`}
          icon={Download}
          label={"Export scope"}
          tone="zinc"
          value={formatNumber(sessions.data?.total ?? 0)}
        />
        <MetricCard
          detail={`${formatNumber(sessions.data?.totals.inputTokens ?? 0)} ${"Input"}`}
          icon={BarChart3}
          label={"Matched tokens"}
          tone="zinc"
          value={formatNumber(sessions.data?.totals.totalTokens ?? 0)}
        />
        <MetricCard
          detail={`${formatNumber(sessions.data?.totals.messages ?? 0)} ${"Messages"}`}
          icon={Bot}
          label={"Tool calls"}
          tone="zinc"
          value={formatNumber(sessions.data?.totals.toolCalls ?? 0)}
        />
        <MetricCard
          detail={"Estimated cost for the current filters"}
          icon={CircleDollarSign}
          label={"Estimated cost"}
          tone="zinc"
          value={formatMoney(sessions.data?.totals.costUsd ?? 0)}
        />
      </section>

      <Panel description={"Download the current filtered results at once."} title={"Bulk export"}>
        <div className="grid gap-3 bg-zinc-50/70 p-4 md:grid-cols-4 xl:grid-cols-7">
          <input
            className={fieldClassName}
            onChange={(event) => {
              const next = new URLSearchParams(searchParams);
              if (event.target.value.trim()) {
                next.set("q", event.target.value.trim());
              } else {
                next.delete("q");
              }
              setSearchParams(next);
            }}
            placeholder={"Search id, cwd, repo"}
            value={query}
          />
          <FacetSelect
            label={"Repository"}
            onChange={(value) => updateFilter("repositoryId", value)}
            options={facets.data?.repositories ?? []}
            value={filters.repositoryId}
          />
          <FacetSelect
            label={"Project"}
            onChange={(value) => updateFilter("projectId", value)}
            options={facets.data?.projects ?? []}
            value={filters.projectId}
          />
          <FacetSelect
            label={"Model"}
            onChange={(value) => updateFilter("model", value)}
            options={facets.data?.models ?? []}
            value={filters.model}
          />
          <BulkExportLink format="csv" query={exportQuery} />
          <BulkExportLink format="json" query={exportQuery} />
          <BulkExportLink format="markdown" query={exportQuery} />
        </div>
      </Panel>

      <Panel
        description={`${sessions.data?.items.length ?? 0} sample sessions`}
        title={"Session exports"}
      >
        <div className="divide-y divide-zinc-100 bg-white">
          {(sessions.data?.items ?? []).map((session) => (
            <div
              className="grid gap-3 px-5 py-3.5 text-sm md:grid-cols-[minmax(0,1fr)_240px] md:items-center"
              key={session.id}
            >
              <div className="min-w-0">
                <Link
                  className="truncate font-medium hover:text-zinc-700"
                  to={`/sessions/${session.id}`}
                >
                  {sessionDisplayTitle(session)}
                </Link>
                {sessionDisplaySummary(session) ? (
                  <div className="mt-1 line-clamp-2 text-xs text-zinc-500">
                    {sessionDisplaySummary(session)}
                  </div>
                ) : null}
                <div className="mt-1 truncate text-xs text-zinc-500">
                  {session.cwd}
                  {session.branch ? ` · ${session.branch}` : ""}
                </div>
                <div className="mt-1 truncate font-mono text-[11px] text-zinc-400">
                  {session.id}
                </div>
              </div>
              <ExportLinks sessionId={session.id} />
            </div>
          ))}
        </div>
        {sessions.loading ? <PanelLoading /> : null}
        {!sessions.loading && sessions.error ? <PanelError message={sessions.error} /> : null}
        {!sessions.loading && (sessions.data?.items ?? []).length === 0 ? (
          <EmptyState label={"No sessions to export"} />
        ) : null}
      </Panel>
    </div>
  );
}
