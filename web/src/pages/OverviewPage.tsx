import {
  BarChart3,
  CalendarDays,
  CircleDollarSign,
  Code2,
  Database,
  FilePlus2,
  GitBranch,
  MessageSquareText,
} from "lucide-react";
import { useState } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import {
  Badge,
  ChartLegend,
  EmptyState,
  ErrorBanner,
  MetricCard,
  Panel,
  PanelError,
  PanelLoading,
} from "@/components/common/primitives";
import { DashboardFilterBar } from "@/components/filters/DashboardFilterBar";
import {
  OverviewHealthStrip,
  OverviewHero,
  UsageWindowStrip,
} from "@/components/overview/overview-widgets";
import { useApiData } from "@/lib/api";
import { emptyDashboardFilters } from "@/lib/constants";
import {
  formatCompact,
  formatMoney,
  formatNumber,
  formatPercent,
  formatShortDate,
} from "@/lib/format";
import { buildFilterQuery, sumUsage } from "@/lib/queries";
import type {
  ArchiveStatus,
  DashboardFilters,
  Facets,
  UsageBreakdown,
  UsageGlobalTotals,
  UsageSeriesPoint,
  UsageSummary,
  UsageWindow,
} from "@/types";

export function OverviewPage() {
  const usageWindows = useApiData<UsageWindow[]>("/api/usage/windows");
  const summary = useApiData<UsageGlobalTotals>("/api/usage/totals");
  const facets = useApiData<Facets>("/api/filter-options");
  const archive = useApiData<ArchiveStatus>("/api/archive/status");
  const [filters, setFilters] = useState<DashboardFilters>(emptyDashboardFilters);
  const [bucket, setBucket] = useState("day");
  const [visibleUsageSeries, setVisibleUsageSeries] = useState({
    input: true,
    output: true,
  });

  const filterQuery = buildFilterQuery(filters);
  const usagePath = `/api/usage/series?bucket=${bucket}${filterQuery ? `&${filterQuery}` : ""}`;
  const summaryPath = `/api/usage/summary${filterQuery ? `?${filterQuery}` : ""}`;
  const languagePath = `/api/usage/breakdown?groupBy=language&limit=8${
    filterQuery ? `&${filterQuery}` : ""
  }`;
  const usage = useApiData<UsageSeriesPoint[]>(usagePath);
  const usageSummary = useApiData<UsageSummary>(summaryPath);
  const languageBreakdown = useApiData<UsageBreakdown[]>(languagePath);

  const chartData = (usage.data ?? []).map((item) => ({
    ...item,
    day: formatShortDate(item.bucket),
    input: item.inputTokens,
    output: item.outputTokens + item.reasoningOutputTokens,
    patchAddedLines: item.patchAddedLines,
  }));
  const totals = usageSummary.data?.current ?? sumUsage(usage.data ?? []);
  const total = totals.totalTokens || summary.data?.totalTokens || 0;
  const inputTokens = totals.inputTokens || summary.data?.inputTokens || 0;
  const outputTokens = totals.outputTokens || summary.data?.outputTokens || 0;
  const reasoningTokens = totals.reasoningOutputTokens;
  const cost = totals.costUsd || summary.data?.costUsd || 0;
  const inputShare = total > 0 ? Math.round((inputTokens / total) * 100) : 0;
  const outputShare = total > 0 ? Math.round(((outputTokens + reasoningTokens) / total) * 100) : 0;
  const overviewError =
    summary.error ||
    usageSummary.error ||
    usage.error ||
    languageBreakdown.error ||
    usageWindows.error ||
    archive.error;
  const languageItems = languageBreakdown.data ?? [];
  const maxLanguageLines = Math.max(...languageItems.map((item) => item.patchAddedLines), 0);

  return (
    <div className="space-y-6">
      <OverviewHero
        devices={summary.data?.devices ?? 0}
        projects={summary.data?.projects ?? 0}
        sessions={summary.data?.sessions ?? 0}
        totalTokens={total}
      />

      {overviewError ? <ErrorBanner message={overviewError} /> : null}

      <OverviewHealthStrip
        activeDays={usageSummary.data?.activeDays ?? 0}
        archive={archive.data}
        facets={facets.data}
        summary={summary.data}
      />

      <UsageWindowStrip windows={usageWindows.data} />

      <DashboardFilterBar
        bucket={bucket}
        facets={facets.data}
        filters={filters}
        onBucketChange={setBucket}
        onFiltersChange={setFilters}
        showBucket
      />

      <section className="hidden gap-3 md:grid md:grid-cols-2 xl:grid-cols-3">
        <MetricCard
          detail={`${inputShare}% ${"Input"} / ${outputShare}% ${"Output"}`}
          icon={BarChart3}
          label={"Total tokens"}
          tone="zinc"
          value={formatNumber(total)}
        />
        <MetricCard
          detail={"Total for the current filters"}
          icon={MessageSquareText}
          label={"Input tokens"}
          tone="zinc"
          value={formatNumber(inputTokens)}
        />
        <MetricCard
          detail={`${formatNumber(reasoningTokens)} reasoning`}
          icon={GitBranch}
          label={"Output tokens"}
          tone="zinc"
          value={formatNumber(outputTokens)}
        />
        <MetricCard
          detail={"Based on the pricing snapshot"}
          icon={CircleDollarSign}
          label={"Estimated cost"}
          tone="zinc"
          value={formatMoney(cost)}
        />
        <MetricCard
          detail={`${formatNumber(inputTokens)} ${"Input"}`}
          icon={Database}
          label={"Cache hit rate"}
          tone="zinc"
          value={formatPercent(usageSummary.data?.cacheHitRate ?? 0)}
        />
        <MetricCard
          detail="apply_patch"
          icon={FilePlus2}
          label={"Patch added lines"}
          tone="zinc"
          value={formatNumber(totals.patchAddedLines ?? 0)}
        />
      </section>

      <section className="grid gap-6">
        <Panel
          action={
            <Badge tone="zinc">
              <CalendarDays size={14} />
              {bucket}
            </Badge>
          }
          description={"Input/output token trend for the selected filters."}
          title={"Usage trend"}
        >
          <div className="px-2 pb-4">
            <div className="flex flex-wrap gap-3 px-4 pt-4">
              <ChartLegend
                active={visibleUsageSeries.input}
                color="bg-zinc-950"
                label={"Input"}
                onClick={() =>
                  setVisibleUsageSeries((current) => ({ ...current, input: !current.input }))
                }
                value={formatNumber(inputTokens)}
              />
              <ChartLegend
                active={visibleUsageSeries.output}
                color="bg-sky-500"
                label={"Output"}
                onClick={() =>
                  setVisibleUsageSeries((current) => ({ ...current, output: !current.output }))
                }
                value={formatNumber(outputTokens + reasoningTokens)}
              />
            </div>
          </div>
          <div className="h-[330px] w-full min-w-0 overflow-hidden px-2 pb-3">
            {usage.loading ? <PanelLoading /> : null}
            {!usage.loading && usage.error ? <PanelError message={usage.error} /> : null}
            {!usage.loading && chartData.length === 0 ? (
              <EmptyState label={"No usage data"} />
            ) : null}
            {!usage.loading && !usage.error && chartData.length > 0 ? (
              <ResponsiveContainer
                className="min-w-0"
                height={300}
                minHeight={300}
                minWidth={0}
                width="100%"
              >
                <LineChart data={chartData} margin={{ bottom: 0, left: 4, right: 18, top: 18 }}>
                  <CartesianGrid stroke="#e5e7eb" strokeDasharray="3 3" vertical={false} />
                  <XAxis
                    axisLine={false}
                    dataKey="day"
                    minTickGap={24}
                    tick={{ fill: "#71717a", fontSize: 12 }}
                    tickLine={false}
                  />
                  <YAxis
                    axisLine={false}
                    tick={{ fill: "#71717a", fontSize: 12 }}
                    tickFormatter={(value) => formatCompact(value as number)}
                    tickLine={false}
                    width={48}
                  />
                  <Tooltip
                    contentStyle={{
                      border: "1px solid #e4e4e7",
                      borderRadius: 0,
                      boxShadow: "none",
                    }}
                    formatter={(value, name) => [
                      formatNumber(value as number),
                      name === "input" ? "Input" : "Output",
                    ]}
                    labelFormatter={(label) => `Date ${label}`}
                  />
                  {visibleUsageSeries.input ? (
                    <Line
                      dataKey="input"
                      dot={false}
                      isAnimationActive={false}
                      stroke="#18181b"
                      strokeLinecap="butt"
                      strokeLinejoin="miter"
                      strokeWidth={2}
                      type="linear"
                    />
                  ) : null}
                  {visibleUsageSeries.output ? (
                    <Line
                      dataKey="output"
                      dot={false}
                      isAnimationActive={false}
                      stroke="#0ea5e9"
                      strokeLinecap="butt"
                      strokeLinejoin="miter"
                      strokeWidth={2}
                      type="linear"
                    />
                  ) : null}
                </LineChart>
              </ResponsiveContainer>
            ) : null}
          </div>
        </Panel>

        <Panel
          action={
            <Badge tone="zinc">
              <CalendarDays size={14} />
              {bucket}
            </Badge>
          }
          description={"apply_patch added-line trend for the selected filters."}
          title={"Patch added lines"}
        >
          <div className="px-2 pb-4">
            <div className="flex flex-wrap gap-3 px-4 pt-4">
              <ChartLegend
                color="bg-zinc-950"
                label={"Added lines"}
                value={formatNumber(totals.patchAddedLines ?? 0)}
              />
            </div>
          </div>
          <div className="h-[300px] w-full min-w-0 overflow-hidden px-2 pb-3">
            {usage.loading ? <PanelLoading /> : null}
            {!usage.loading && usage.error ? <PanelError message={usage.error} /> : null}
            {!usage.loading && chartData.length === 0 ? (
              <EmptyState label={"No patch data"} />
            ) : null}
            {!usage.loading && !usage.error && chartData.length > 0 ? (
              <ResponsiveContainer
                className="min-w-0"
                height={270}
                minHeight={270}
                minWidth={0}
                width="100%"
              >
                <BarChart data={chartData} margin={{ bottom: 0, left: 4, right: 18, top: 18 }}>
                  <CartesianGrid stroke="#e5e7eb" strokeDasharray="3 3" vertical={false} />
                  <XAxis
                    axisLine={false}
                    dataKey="day"
                    minTickGap={24}
                    tick={{ fill: "#71717a", fontSize: 12 }}
                    tickLine={false}
                  />
                  <YAxis
                    allowDecimals={false}
                    axisLine={false}
                    tick={{ fill: "#71717a", fontSize: 12 }}
                    tickFormatter={(value) => formatCompact(value as number)}
                    tickLine={false}
                    width={48}
                  />
                  <Tooltip
                    contentStyle={{
                      border: "1px solid #e4e4e7",
                      borderRadius: 0,
                      boxShadow: "none",
                    }}
                    formatter={(value) => [formatNumber(value as number), "Added lines"]}
                    labelFormatter={(label) => `Date ${label}`}
                  />
                  <Bar
                    dataKey="patchAddedLines"
                    fill="#18181b"
                    isAnimationActive={false}
                    radius={0}
                  />
                </BarChart>
              </ResponsiveContainer>
            ) : null}
          </div>
        </Panel>

        <Panel
          action={
            <Badge tone="zinc">
              <Code2 size={14} />
              {formatNumber(languageItems.length)}
            </Badge>
          }
          description={"apply_patch added lines by detected file language."}
          title={"Patch languages"}
        >
          <div className="min-h-[260px] px-4 py-4">
            {languageBreakdown.loading ? <PanelLoading /> : null}
            {!languageBreakdown.loading && languageBreakdown.error ? (
              <PanelError message={languageBreakdown.error} />
            ) : null}
            {!languageBreakdown.loading && languageItems.length === 0 ? (
              <EmptyState label={"No language data"} />
            ) : null}
            {!languageBreakdown.loading && !languageBreakdown.error && languageItems.length > 0 ? (
              <div className="space-y-3">
                {languageItems.map((item) => {
                  const width =
                    maxLanguageLines > 0
                      ? `${Math.max(5, (item.patchAddedLines / maxLanguageLines) * 100)}%`
                      : "0%";
                  return (
                    <div key={item.id}>
                      <div className="mb-1.5 flex items-center justify-between gap-3 text-sm">
                        <div className="min-w-0 truncate font-medium text-zinc-800">
                          {item.label}
                        </div>
                        <div className="shrink-0 text-zinc-500 text-xs tabular-nums">
                          {formatNumber(item.patchAddedLines)} {"lines"}
                        </div>
                      </div>
                      <div className="h-2 overflow-hidden bg-zinc-100">
                        <div className="h-full bg-zinc-950" style={{ width }} />
                      </div>
                      <div className="mt-1 text-[11px] text-zinc-400">
                        {formatNumber(item.sessions)} {"sessions"}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : null}
          </div>
        </Panel>
      </section>
    </div>
  );
}
