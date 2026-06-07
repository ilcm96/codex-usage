import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Filter, Search } from "lucide-react";
import type { FormEvent } from "react";
import { useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { HighlightedText } from "@/components/common/data-display";
import {
  Badge,
  EmptyState,
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
import { formatDate, formatNumber, pathTail } from "@/lib/format";
import { buildSearchQuery, filtersFromSearchParams } from "@/lib/queries";
import type { DashboardFilters, Facets, SearchResponse } from "@/types";

export function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const facets = useApiData<Facets>("/api/filter-options");
  const initialQuery = searchParams.get("q") ?? "";
  const [query, setQuery] = useState(initialQuery);
  const filters = filtersFromSearchParams(searchParams);
  const kind = searchParams.get("kind") ?? "message";
  const offset = Number(searchParams.get("offset") ?? "0");
  const resultsQuery = buildSearchQuery(filters, initialQuery, kind, 25, offset);
  const results = useApiData<SearchResponse>(initialQuery ? `/api/search?${resultsQuery}` : "");
  const resultItems = results.data?.items ?? [];
  const [filtersOpen, setFiltersOpen] = useState(false);
  const activeFilters = Object.values(filters).filter(Boolean).length + (kind ? 1 : 0);
  const resultCountLabel = results.loading
    ? "Searching..."
    : results.data
      ? `${formatNumber(results.data.total)}${results.data.totalKnown === false ? "+" : ""} results`
      : "0 results";

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextQuery = query.trim();
    const next = new URLSearchParams(searchParams);
    if (nextQuery) {
      next.set("q", nextQuery);
    } else {
      next.delete("q");
    }
    next.delete("offset");
    setSearchParams(next);
  }

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

  function moveSearchPage(nextOffset: number) {
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
        eyebrow={"Search"}
        title={"Full-text search"}
        description={
          "By default this searches user and assistant messages. Extend to tool/raw logs when needed."
        }
      />

      <Panel
        action={
          <div className="flex flex-wrap items-center gap-2">
            {initialQuery ? (
              <>
                <Button
                  disabled={offset <= 0}
                  onClick={() => moveSearchPage(Math.max(0, offset - 50))}
                  size="sm"
                  type="button"
                  variant="outline"
                >
                  <ChevronLeft size={14} />
                  {"Prev"}
                </Button>
                <Button
                  disabled={!results.data?.nextOffset}
                  onClick={() => moveSearchPage(results.data?.nextOffset ?? 0)}
                  size="sm"
                  type="button"
                  variant="outline"
                >
                  {"Next"}
                  <ChevronRight size={14} />
                </Button>
              </>
            ) : null}
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
          </div>
        }
        description={resultCountLabel}
        title={"Search"}
      >
        <form
          className="grid gap-2 border-zinc-200 border-b bg-zinc-50/70 p-3 md:grid-cols-[minmax(0,1fr)_120px]"
          onSubmit={submit}
        >
          <input
            className="h-10 min-w-0 flex-1 rounded-none border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-zinc-400 focus:ring-4 focus:ring-zinc-100"
            onChange={(event) => setQuery(event.target.value)}
            placeholder={"Search conversations and tool outputs"}
            value={query}
          />
          <Button className="h-10" type="submit">
            <Search size={15} />
            {"Search"}
          </Button>
        </form>
        <div
          aria-hidden={!filtersOpen}
          className={`grid gap-2 border-zinc-200 border-b bg-zinc-50/70 p-3 md:grid-cols-[repeat(5,minmax(0,1fr))] ${
            filtersOpen ? "grid" : "hidden"
          }`}
          hidden={!filtersOpen}
        >
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
          <DatePicker
            onChange={(value) => updateFilter("from", value)}
            placeholder={"From date"}
            value={filters.from}
          />
          <Select
            onValueChange={(value) => {
              const next = new URLSearchParams(searchParams);
              if (value) {
                next.set("kind", value);
              } else {
                next.delete("kind");
              }
              next.delete("offset");
              setSearchParams(next);
            }}
            options={[
              { label: "Messages", value: "message" },
              { label: "User only", value: "user" },
              { label: "Assistant only", value: "assistant" },
              { label: "Tools", value: "tool" },
              { label: "All documents", value: "all" },
            ]}
            placeholder={"Event kind"}
            value={kind}
          />
        </div>
        {results.loading ? <PanelLoading label={"Searching conversations..."} /> : null}
        {!results.loading && results.error ? <PanelError message={results.error} /> : null}
        <div className="divide-y divide-zinc-100 bg-white">
          {resultItems.map((result) => (
            <Link
              className="block p-4 transition hover:bg-zinc-50"
              key={`${result.sessionId}-${result.kind}-${result.seq}`}
              to={`/sessions/${encodeURIComponent(result.sessionId)}`}
            >
              <div className="mb-2 flex flex-wrap items-center gap-2 text-xs text-zinc-500">
                <Badge tone="zinc">{result.documentScope || result.kind}</Badge>
                <span>{result.title || result.role || result.toolName || "event"}</span>
                <span>#{result.seq}</span>
                {result.turnIndex != null ? (
                  <span>
                    {"turns"} {result.turnIndex + 1}
                  </span>
                ) : null}
                <span>{result.repository || "local"}</span>
                <span className="truncate">{result.project || pathTail(result.cwd)}</span>
                <span className="ml-auto">{formatDate(result.occurredAt)}</span>
              </div>
              <div className="mb-2 truncate font-medium text-sm">
                {result.sessionTitle || result.project || result.repository || pathTail(result.cwd)}
              </div>
              <p className="line-clamp-4 break-words text-sm text-zinc-800">
                <HighlightedText
                  end={result.matchEnd}
                  start={result.matchStart}
                  text={result.snippet || result.text}
                />
              </p>
              {result.sessionSummary ? (
                <div className="mt-2 line-clamp-2 text-xs text-zinc-500">
                  {result.sessionSummary}
                </div>
              ) : null}
              <div className="mt-2 truncate font-mono text-[11px] text-zinc-400">{result.cwd}</div>
            </Link>
          ))}
        </div>
        {!results.loading && initialQuery && resultItems.length === 0 ? (
          <EmptyState label={"No search results"} />
        ) : null}
        {!initialQuery ? (
          <EmptyState label={"Enter a query to search messages and tool events together."} />
        ) : null}
      </Panel>
    </div>
  );
}
