import {
  BarChart3,
  Bot,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  FilePlus2,
  Filter,
  MessageSquareText,
  Search,
} from "lucide-react";
import type { FormEvent } from "react";
import { useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import {
  Badge,
  EmptyState,
  MetricCard,
  PageTitle,
  Panel,
  PanelError,
  PanelLoading,
} from "@/components/common/primitives";
import { FacetSelect } from "@/components/filters/DashboardFilterBar";
import { Button } from "@/components/ui/button";
import { DatePicker } from "@/components/ui/date-picker";
import { Select } from "@/components/ui/select";
import { useApiData } from "@/lib/api";
import { fieldClassName } from "@/lib/constants";
import {
  formatDate,
  formatMoney,
  formatNumber,
  pathTail,
  sessionDisplaySummary,
  sessionDisplayTitle,
} from "@/lib/format";
import { buildSessionQuery, filtersFromSearchParams } from "@/lib/queries";
import type { DashboardFilters, Facets, SessionsListResponse } from "@/types";

export function SessionsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const facets = useApiData<Facets>("/api/filter-options");
  const [query, setQuery] = useState(searchParams.get("q") ?? "");
  const filters = filtersFromSearchParams(searchParams);
  const sort = searchParams.get("sort") ?? "updated_desc";
  const offset = Number(searchParams.get("offset") ?? "0");
  const sessionsQuery = buildSessionQuery(filters, query, sort, 50, offset);
  const sessions = useApiData<SessionsListResponse>(`/api/sessions?${sessionsQuery}`);
  const sessionItems = sessions.data?.items ?? [];
  const [filtersOpen, setFiltersOpen] = useState(false);
  const activeFilters = Object.values(filters).filter(Boolean).length + (query.trim() ? 1 : 0);

  function updateFilter(key: keyof DashboardFilters, value: string) {
    const next = new URLSearchParams(searchParams);
    if (value) {
      next.set(key, value);
    } else {
      next.delete(key);
    }
    next.delete("offset");
    setSearchParams(next);
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const next = new URLSearchParams(searchParams);
    const trimmed = query.trim();
    if (trimmed) {
      next.set("q", trimmed);
    } else {
      next.delete("q");
    }
    next.delete("offset");
    setSearchParams(next);
  }

  function movePage(nextOffset: number) {
    const next = new URLSearchParams(searchParams);
    if (nextOffset > 0) {
      next.set("offset", String(nextOffset));
    } else {
      next.delete("offset");
    }
    setSearchParams(next);
  }

  return (
    <div className="space-y-6">
      <PageTitle
        eyebrow={"Sessions"}
        title={"Conversation list"}
        description={"Explore sessions by repo, cwd, device, and token usage."}
      />

      <Panel
        action={
          <Button
            onClick={() => setFiltersOpen((value) => !value)}
            size="sm"
            type="button"
            variant="outline"
          >
            <Filter size={14} />
            {activeFilters ? `${activeFilters} active` : "Filters"}
            {filtersOpen ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          </Button>
        }
        description={"List filters and sorting."}
        title={"Session filters"}
      >
        <form
          aria-hidden={!filtersOpen}
          className={`gap-3 bg-zinc-50/70 p-4 md:grid-cols-[minmax(0,1.5fr)_repeat(3,minmax(0,1fr))_160px] xl:grid-cols-[minmax(0,1.6fr)_repeat(6,minmax(0,1fr))] ${
            filtersOpen ? "grid" : "hidden"
          }`}
          hidden={!filtersOpen}
          onSubmit={submit}
        >
          <input
            className={fieldClassName}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={"Search id, cwd, repo, branch"}
            value={query}
          />
          <DatePicker
            onChange={(value) => updateFilter("from", value)}
            placeholder={"From date"}
            value={filters.from}
          />
          <DatePicker
            onChange={(value) => updateFilter("to", value)}
            placeholder={"To date"}
            value={filters.to}
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
          <Select
            onValueChange={(value) => {
              const next = new URLSearchParams(searchParams);
              next.set("sort", value);
              next.delete("offset");
              setSearchParams(next);
            }}
            options={[
              { label: "Updated desc", value: "updated_desc" },
              { label: "Started desc", value: "started_desc" },
              { label: "Tokens desc", value: "tokens_desc" },
              { label: "Tokens asc", value: "tokens_asc" },
              { label: "Cost desc", value: "cost_desc" },
              { label: "Cost asc", value: "cost_asc" },
            ]}
            placeholder={"Sort"}
            value={sort}
          />
          <Button className="h-10" type="submit">
            <Search size={15} />
            {"Apply"}
          </Button>
        </form>
      </Panel>

      <section className="hidden gap-3 md:grid md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          detail={`${sessionItems.length} shown`}
          icon={MessageSquareText}
          label={"Matched sessions"}
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
          detail={`${formatNumber(sessions.data?.totals.toolCalls ?? 0)} ${"Tools"}`}
          icon={Bot}
          label={"Messages"}
          tone="zinc"
          value={formatNumber(sessions.data?.totals.messages ?? 0)}
        />
        <MetricCard
          detail="apply_patch"
          icon={FilePlus2}
          label={"Added lines"}
          tone="zinc"
          value={formatNumber(sessions.data?.totals.patchAddedLines ?? 0)}
        />
      </section>

      <Panel
        action={
          <div className="flex items-center gap-2">
            <Button
              disabled={offset <= 0}
              onClick={() => movePage(Math.max(0, offset - 50))}
              size="sm"
              type="button"
              variant="outline"
            >
              <ChevronLeft size={14} />
              {"Prev"}
            </Button>
            <Button
              disabled={!sessions.data?.nextOffset}
              onClick={() => movePage(sessions.data?.nextOffset ?? 0)}
              size="sm"
              type="button"
              variant="outline"
            >
              {"Next"}
              <ChevronRight size={14} />
            </Button>
          </div>
        }
        description={`${formatNumber(offset + 1)}-${formatNumber(offset + sessionItems.length)} / ${formatNumber(sessions.data?.total ?? 0)}`}
        title={"All sessions"}
      >
        <div className="overflow-hidden">
          <div className="hidden grid-cols-[minmax(0,1.45fr)_minmax(0,0.9fr)_105px_90px_90px_90px_120px] border-zinc-200 border-b bg-zinc-50/70 px-5 py-2.5 font-medium text-[11px] text-zinc-500 uppercase tracking-wide md:grid">
            <div>{"Session"}</div>
            <div>{"Repository"}</div>
            <div className="text-right">{"Total tokens"}</div>
            <div className="text-right">{"Cost"}</div>
            <div className="text-right">{"Events"}</div>
            <div className="text-right">Patch +</div>
            <div className="text-right">{"Updated"}</div>
          </div>
          <div className="divide-y divide-zinc-100 bg-white">
            {sessionItems.map((session) => (
              <Link
                className="grid gap-2 px-4 py-3.5 text-sm transition hover:bg-zinc-50 md:grid-cols-[minmax(0,1.45fr)_minmax(0,0.9fr)_105px_90px_90px_90px_120px] md:items-center md:gap-3 md:px-5"
                key={session.id}
                to={`/sessions/${encodeURIComponent(session.id)}`}
              >
                <div className="min-w-0">
                  <div className="line-clamp-2 font-medium md:truncate">
                    {sessionDisplayTitle(session)}
                  </div>
                  <div className="mt-1 truncate text-xs text-zinc-500">
                    {session.repository || "local"} · {session.project || pathTail(session.cwd)}
                    {session.branch ? ` · ${session.branch}` : ""}
                  </div>
                  {sessionDisplaySummary(session) ? (
                    <div className="mt-1 line-clamp-2 text-[11px] text-zinc-500 md:line-clamp-1">
                      {sessionDisplaySummary(session)}
                    </div>
                  ) : null}
                  <div className="mt-2 flex flex-wrap gap-1.5 md:hidden">
                    <Badge tone="zinc">{formatNumber(session.totalTokens)}</Badge>
                    <Badge tone="zinc">{formatMoney(session.costUsd)}</Badge>
                    <Badge tone="zinc">
                      {formatNumber(session.conversationTurns ?? 0)} {"turns"}
                    </Badge>
                    <Badge tone="zinc">{formatDate(session.updatedAt ?? session.startedAt)}</Badge>
                  </div>
                </div>
                <div className="hidden truncate text-zinc-600 md:block">
                  <Badge tone="zinc">{session.repository || "local"}</Badge>
                  <div className="mt-1 truncate text-[11px] text-zinc-400">
                    {session.displaySubtitle || session.cwd}
                  </div>
                </div>
                <div className="hidden font-medium tabular-nums md:block md:text-right">
                  {formatNumber(session.totalTokens)}
                </div>
                <div className="hidden text-zinc-600 text-xs tabular-nums md:block md:text-right">
                  {formatMoney(session.costUsd)}
                </div>
                <div className="hidden text-zinc-500 text-xs tabular-nums md:block md:text-right">
                  {formatNumber(session.conversationTurns ?? 0)} {"turns"}
                </div>
                <div className="hidden text-zinc-500 text-xs tabular-nums md:block md:text-right">
                  {formatNumber(session.patchAddedLines ?? 0)}
                </div>
                <div className="hidden text-zinc-500 text-xs md:block md:text-right">
                  {formatDate(session.updatedAt ?? session.startedAt)}
                </div>
              </Link>
            ))}
          </div>
          {sessions.loading ? <PanelLoading /> : null}
          {!sessions.loading && sessions.error ? <PanelError message={sessions.error} /> : null}
          {!sessions.loading && sessionItems.length === 0 ? (
            <EmptyState label={"No sessions"} />
          ) : null}
        </div>
      </Panel>
    </div>
  );
}
