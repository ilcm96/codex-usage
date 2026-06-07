import {
  Activity,
  CalendarDays,
  CircleDollarSign,
  Database,
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

import { CalendarHeatmap } from "@/components/common/data-display";
import {
  Badge,
  ChartLegend,
  EmptyState,
  ErrorBanner,
  MetricCard,
  PageTitle,
  Panel,
  PanelError,
  PanelLoading,
  UsageRow,
} from "@/components/common/primitives";
import { DashboardFilterBar } from "@/components/filters/DashboardFilterBar";
import { buildProjectChartData, ProjectRankingTooltip } from "@/components/overview/project-chart";
import { useApiData } from "@/lib/api";
import { emptyDashboardFilters } from "@/lib/constants";
import {
  formatCompact,
  formatMoney,
  formatNumber,
  formatPercent,
  formatShortDate,
  truncateMiddle,
} from "@/lib/format";
import { buildFilterQuery, sumUsage } from "@/lib/queries";
import type {
  DashboardFilters,
  Facets,
  Project,
  UsageBreakdown,
  UsageCalendarDay,
  UsageSeriesPoint,
  UsageSummary,
} from "@/types";

export function UsagePage() {
  const facets = useApiData<Facets>("/api/filter-options");
  const projects = useApiData<Project[]>("/api/projects");
  const [filters, setFilters] = useState<DashboardFilters>(emptyDashboardFilters);
  const [bucket, setBucket] = useState("day");
  const [breakdownGroup, setBreakdownGroup] = useState("model");
  const [visibleTrendSeries, setVisibleTrendSeries] = useState({
    cost: true,
    tokens: true,
  });

  const filterQuery = buildFilterQuery(filters);
  const usagePath = `/api/usage/series?bucket=${bucket}${filterQuery ? `&${filterQuery}` : ""}`;
  const summaryPath = `/api/usage/summary${filterQuery ? `?${filterQuery}` : ""}`;
  const breakdownPath = `/api/usage/breakdown?groupBy=${breakdownGroup}&limit=12${
    filterQuery ? `&${filterQuery}` : ""
  }`;
  const repositoryBreakdownPath = `/api/usage/breakdown?groupBy=repository&limit=8${
    filterQuery ? `&${filterQuery}` : ""
  }`;
  const calendarPath = `/api/usage/calendar?days=365${filterQuery ? `&${filterQuery}` : ""}`;

  const usage = useApiData<UsageSeriesPoint[]>(usagePath);
  const usageSummary = useApiData<UsageSummary>(summaryPath);
  const breakdown = useApiData<UsageBreakdown[]>(breakdownPath);
  const repositoryBreakdown = useApiData<UsageBreakdown[]>(repositoryBreakdownPath);
  const calendar = useApiData<UsageCalendarDay[]>(calendarPath);

  const usageTotals = usageSummary.data?.current ?? sumUsage(usage.data ?? []);
  const totalTokens = usageTotals.totalTokens;
  const averageSessionTokens =
    usageTotals.sessions > 0 ? Math.round(totalTokens / usageTotals.sessions) : 0;
  const chartData = (usage.data ?? []).map((item) => ({
    ...item,
    cost: item.costUsd,
    day: formatShortDate(item.bucket),
    tokens: item.totalTokens,
  }));
  const projectChartData = buildProjectChartData(projects.data ?? [], 8);
  const projectLabelByKey = new Map(
    projectChartData.map((project) => [project.chartKey, project.chartLabel]),
  );
  const usageError =
    facets.error ||
    projects.error ||
    usage.error ||
    usageSummary.error ||
    breakdown.error ||
    repositoryBreakdown.error ||
    calendar.error;
  const usageGroups = [
    { label: "Model", value: "model" },
    { label: "Repository", value: "repository" },
    { label: "Project", value: "project" },
    { label: "Device", value: "device" },
  ];

  return (
    <div className="space-y-6">
      <PageTitle
        description={"Analyze tokens, cost, models, repositories, and session usage by filter."}
        eyebrow={"Usage"}
        title={"Usage analytics"}
      />

      {usageError ? <ErrorBanner message={usageError} /> : null}

      <DashboardFilterBar
        bucket={bucket}
        facets={facets.data}
        filters={filters}
        onBucketChange={setBucket}
        onFiltersChange={setFilters}
        showBucket
      />

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          detail={"Total for the current filters"}
          icon={Activity}
          label={"Current tokens"}
          tone="zinc"
          value={formatNumber(totalTokens)}
        />
        <MetricCard
          detail={"Estimated cost for the current filters"}
          icon={CircleDollarSign}
          label={"Estimated cost"}
          tone="zinc"
          value={formatMoney(usageTotals.costUsd)}
        />
        <MetricCard
          detail={`${formatNumber(usageTotals.sessions)} ${"Sessions"}`}
          icon={MessageSquareText}
          label={"Avg session tokens"}
          tone="zinc"
          value={formatNumber(averageSessionTokens)}
        />
        <MetricCard
          detail="cached input / input"
          icon={Database}
          label={"Cache hit rate"}
          tone="zinc"
          value={formatPercent(usageSummary.data?.cacheHitRate ?? 0)}
        />
      </section>

      <section>
        <Panel
          action={
            <SegmentedControl
              onChange={setBreakdownGroup}
              options={usageGroups}
              value={breakdownGroup}
            />
          }
          description={"Review top usage areas and concentration for the selected filters."}
          title={"Breakdown"}
        >
          <div className="space-y-4 p-4">
            {(breakdown.data ?? []).map((item) => (
              <UsageRow
                detail={`${formatNumber(item.sessions)} ${"Sessions"} · ${formatMoney(item.costUsd)}`}
                key={`${breakdownGroup}-${item.id}-${item.label}`}
                label={item.label || "unknown"}
                max={maxTokens(breakdown.data)}
                value={item.totalTokens}
              />
            ))}
            {breakdown.loading ? <PanelLoading /> : null}
            {!breakdown.loading && breakdown.error ? (
              <PanelError message={breakdown.error} />
            ) : null}
            {!breakdown.loading && (breakdown.data ?? []).length === 0 ? (
              <EmptyState label={"No analysis data"} />
            ) : null}
          </div>
        </Panel>
      </section>

      <section>
        <Panel
          action={
            <Badge tone="zinc">
              <CalendarDays size={14} />
              {bucket}
            </Badge>
          }
          description={"Compare token volume and cost over the same period."}
          title={"Cost and token trend"}
        >
          <div className="px-2 pb-4">
            <div className="flex flex-wrap gap-3 px-4 pt-4">
              <ChartLegend
                active={visibleTrendSeries.tokens}
                color="bg-zinc-950"
                label={"Total tokens"}
                onClick={() =>
                  setVisibleTrendSeries((current) => ({ ...current, tokens: !current.tokens }))
                }
                value={formatNumber(totalTokens)}
              />
              <ChartLegend
                active={visibleTrendSeries.cost}
                color="bg-sky-500"
                label={"Cost"}
                onClick={() =>
                  setVisibleTrendSeries((current) => ({ ...current, cost: !current.cost }))
                }
                value={formatMoney(usageTotals.costUsd)}
              />
            </div>
          </div>
          <div className="h-[300px] w-full min-w-0 overflow-hidden px-2 pb-3">
            {usage.loading ? <PanelLoading /> : null}
            {!usage.loading && usage.error ? <PanelError message={usage.error} /> : null}
            {!usage.loading && chartData.length === 0 ? (
              <EmptyState label={"No usage data"} />
            ) : null}
            {!usage.loading && !usage.error && chartData.length > 0 ? (
              <ResponsiveContainer
                className="min-w-0"
                height={270}
                minHeight={270}
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
                    yAxisId="tokens"
                  />
                  <YAxis hide orientation="right" yAxisId="cost" />
                  <Tooltip
                    contentStyle={{
                      border: "1px solid #e4e4e7",
                      borderRadius: 0,
                      boxShadow: "none",
                    }}
                    formatter={(value, name) => [
                      name === "cost"
                        ? formatMoney(value as number)
                        : formatNumber(value as number),
                      name === "cost" ? "Cost" : "Total tokens",
                    ]}
                    labelFormatter={(label) => `Date ${label}`}
                  />
                  {visibleTrendSeries.tokens ? (
                    <Line
                      dataKey="tokens"
                      dot={false}
                      isAnimationActive={false}
                      stroke="#18181b"
                      strokeLinecap="butt"
                      strokeLinejoin="miter"
                      strokeWidth={2}
                      type="linear"
                      yAxisId="tokens"
                    />
                  ) : null}
                  {visibleTrendSeries.cost ? (
                    <Line
                      dataKey="cost"
                      dot={false}
                      isAnimationActive={false}
                      stroke="#0ea5e9"
                      strokeLinecap="butt"
                      strokeLinejoin="miter"
                      strokeWidth={2}
                      type="linear"
                      yAxisId="cost"
                    />
                  ) : null}
                </LineChart>
              </ResponsiveContainer>
            ) : null}
          </div>
        </Panel>
      </section>

      <section>
        <Panel description={"Token usage density over the last 365 days."} title={"Usage calendar"}>
          <CalendarHeatmap items={calendar.data ?? []} loading={calendar.loading} />
        </Panel>
      </section>

      <section>
        <Panel description={"Top cwd entries by token usage."} title={"Project ranking"}>
          <div className="h-[320px] w-full min-w-0 overflow-hidden px-2 pb-3 pt-4">
            {projects.loading ? <PanelLoading /> : null}
            {!projects.loading && projects.error ? <PanelError message={projects.error} /> : null}
            {!projects.loading && projectChartData.length === 0 ? (
              <EmptyState label={"Project"} />
            ) : null}
            {!projects.loading && !projects.error && projectChartData.length > 0 ? (
              <ResponsiveContainer
                className="min-w-0"
                height={280}
                minHeight={280}
                minWidth={0}
                width="100%"
              >
                <BarChart data={projectChartData} layout="vertical" margin={{ left: 8, right: 18 }}>
                  <CartesianGrid horizontal={false} stroke="#e5e7eb" strokeDasharray="3 3" />
                  <XAxis
                    axisLine={false}
                    tick={{ fill: "#71717a", fontSize: 12 }}
                    tickFormatter={(value) => formatCompact(value as number)}
                    tickLine={false}
                    type="number"
                  />
                  <YAxis
                    axisLine={false}
                    dataKey="chartKey"
                    tick={{ fill: "#52525b", fontSize: 12 }}
                    tickFormatter={(value) =>
                      truncateMiddle(projectLabelByKey.get(String(value)) ?? String(value), 22)
                    }
                    tickLine={false}
                    type="category"
                    width={132}
                  />
                  <Tooltip
                    content={<ProjectRankingTooltip />}
                    contentStyle={{
                      border: "1px solid #e4e4e7",
                      borderRadius: 0,
                      boxShadow: "none",
                    }}
                    formatter={(value) => [formatNumber(value as number), "Total tokens"]}
                  />
                  <Bar dataKey="totalTokens" fill="#18181b" radius={[0, 0, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            ) : null}
          </div>
        </Panel>
      </section>

      <section>
        <Panel
          description={"Review repository usage by cost and concentration."}
          title={"Repository operating map"}
        >
          <div className="grid gap-3 p-4 md:grid-cols-2 xl:grid-cols-4">
            {(repositoryBreakdown.data ?? []).map((item) => (
              <RepositorySignalCard
                item={item}
                key={`repository-signal-${item.id}-${item.label}`}
                totalTokens={totalTokens}
              />
            ))}
            {repositoryBreakdown.loading ? <PanelLoading /> : null}
            {!repositoryBreakdown.loading && repositoryBreakdown.error ? (
              <PanelError message={repositoryBreakdown.error} />
            ) : null}
            {!repositoryBreakdown.loading && (repositoryBreakdown.data ?? []).length === 0 ? (
              <EmptyState label={"No repository data"} />
            ) : null}
          </div>
        </Panel>
      </section>
    </div>
  );
}

function SegmentedControl({
  onChange,
  options,
  value,
}: {
  onChange: (value: string) => void;
  options: Array<{ label: string; value: string }>;
  value: string;
}) {
  return (
    <div className="flex flex-wrap gap-1">
      {options.map((option) => (
        <button
          className={`h-7 rounded-none border px-2 font-medium text-[11px] uppercase ${
            value === option.value
              ? "border-zinc-950 bg-zinc-950 text-white"
              : "border-zinc-200 bg-white text-zinc-600 hover:bg-zinc-50"
          }`}
          key={option.value}
          onClick={() => onChange(option.value)}
          type="button"
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

function RepositorySignalCard({
  item,
  totalTokens,
}: {
  item: UsageBreakdown;
  totalTokens: number;
}) {
  const share = totalTokens > 0 ? item.totalTokens / totalTokens : 0;
  const avgCostPerSession = item.sessions > 0 ? item.costUsd / item.sessions : 0;

  return (
    <article className="min-w-0 rounded-none border border-zinc-200 bg-zinc-50 p-3">
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium text-sm">{item.label || "unknown"}</div>
          <div className="mt-1 truncate text-xs text-zinc-500">{item.detail || "Repository"}</div>
        </div>
        <Badge tone="zinc">{formatPercent(share)}</Badge>
      </div>
      <div className="mt-4 h-2 rounded-none bg-white">
        <div
          className="h-2 rounded-none bg-zinc-950"
          style={{ width: `${Math.max(3, share * 100)}%` }}
        />
      </div>
      <div className="mt-4 grid grid-cols-2 gap-3 text-xs">
        <div>
          <div className="text-zinc-400">{"Total tokens"}</div>
          <div className="font-medium tabular-nums">{formatNumber(item.totalTokens)}</div>
        </div>
        <div>
          <div className="text-zinc-400">{"Cost/session"}</div>
          <div className="font-medium tabular-nums">{formatMoney(avgCostPerSession)}</div>
        </div>
        <div>
          <div className="text-zinc-400">{"Sessions"}</div>
          <div className="font-medium tabular-nums">{formatNumber(item.sessions)}</div>
        </div>
        <div>
          <div className="text-zinc-400">{"Total cost"}</div>
          <div className="font-medium tabular-nums">{formatMoney(item.costUsd)}</div>
        </div>
      </div>
    </article>
  );
}

function maxTokens(items: UsageBreakdown[] | null | undefined) {
  return Math.max(...(items ?? []).map((row) => row.totalTokens), 1);
}
