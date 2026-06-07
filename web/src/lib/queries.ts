import type { DashboardFilters, UsageSeriesPoint } from "@/types";

export function buildFilterQuery(filters: DashboardFilters) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value) {
      params.set(key, value);
    }
  }
  return params.toString();
}

export function buildSessionQuery(
  filters: DashboardFilters,
  query: string,
  sort: string,
  limit: number,
  offset = 0,
) {
  const params = new URLSearchParams(buildFilterQuery(filters));
  params.set("limit", String(limit));
  params.set("offset", String(offset));
  params.set("sort", sort);
  if (query.trim()) {
    params.set("q", query.trim());
  }
  return params.toString();
}

export function buildSearchQuery(
  filters: DashboardFilters,
  query: string,
  kind: string,
  limit: number,
  offset: number,
) {
  const params = new URLSearchParams(buildFilterQuery(filters));
  params.set("limit", String(limit));
  params.set("offset", String(offset));
  params.set("q", query.trim());
  params.set("includeTotal", "false");
  if (kind) {
    params.set("kind", kind);
  }
  return params.toString();
}

export function filtersFromSearchParams(params: URLSearchParams): DashboardFilters {
  return {
    deviceId: params.get("deviceId") ?? "",
    from: params.get("from") ?? "",
    model: params.get("model") ?? "",
    projectId: params.get("projectId") ?? "",
    repositoryId: params.get("repositoryId") ?? "",
    to: params.get("to") ?? "",
  };
}

export function sumUsage(items: UsageSeriesPoint[]) {
  return items.reduce(
    (acc, item) => ({
      cachedInputTokens: acc.cachedInputTokens + item.cachedInputTokens,
      costUsd: acc.costUsd + item.costUsd,
      inputTokens: acc.inputTokens + item.inputTokens,
      outputTokens: acc.outputTokens + item.outputTokens,
      patchAddedLines: acc.patchAddedLines + item.patchAddedLines,
      reasoningOutputTokens: acc.reasoningOutputTokens + item.reasoningOutputTokens,
      totalTokens: acc.totalTokens + item.totalTokens,
      messages: 0,
      sessions: 0,
      toolCalls: 0,
    }),
    {
      cachedInputTokens: 0,
      costUsd: 0,
      inputTokens: 0,
      messages: 0,
      outputTokens: 0,
      patchAddedLines: 0,
      reasoningOutputTokens: 0,
      sessions: 0,
      toolCalls: 0,
      totalTokens: 0,
    },
  );
}
