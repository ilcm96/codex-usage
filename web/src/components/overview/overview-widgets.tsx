import type { LucideIcon } from "lucide-react";
import {
  ArrowUpRight,
  CheckCircle2,
  Clock,
  Database,
  FileArchive,
  RefreshCw,
  Search,
} from "lucide-react";
import { Link } from "react-router-dom";

import { Badge } from "@/components/common/primitives";
import {
  formatBytes,
  formatDate,
  formatMoney,
  formatNumber,
  formatPercent,
  formatPlainDate,
} from "@/lib/format";
import type { ArchiveStatus, Facets, UsageGlobalTotals, UsageWindow } from "@/types";

export function OverviewHero({
  totalTokens,
  projects,
  devices,
  sessions,
}: {
  totalTokens: number;
  projects: number;
  devices: number;
  sessions: number;
}) {
  return (
    <section className="overflow-hidden rounded-none border border-zinc-200 bg-white">
      <div className="px-5 py-4 md:px-6 md:py-5">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm">
          <div className="inline-flex items-center gap-1.5 font-medium text-zinc-700">
            <CheckCircle2 size={14} />
            {"Synced"}
          </div>
          <div className="text-zinc-500">
            <span className="font-medium text-zinc-950">{formatNumber(sessions)}</span> {"Sessions"}
          </div>
          <div className="text-zinc-500">
            <span className="font-medium text-zinc-950">{formatNumber(projects)}</span> {"projects"}
          </div>
          <div className="text-zinc-500">
            <span className="font-medium text-zinc-950">{formatNumber(devices)}</span> {"devices"}
          </div>
        </div>

        <div className="mt-3 grid gap-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
          <h2 className="min-w-0 font-semibold text-3xl leading-tight tracking-normal md:whitespace-nowrap">
            {formatNumber(totalTokens)} tokens across {formatNumber(sessions)} {"Sessions"}
          </h2>
          <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row md:justify-self-end">
            <Link
              className="inline-flex h-9 min-w-28 items-center justify-between gap-2 rounded-none bg-zinc-950 px-3 font-medium text-white text-xs transition hover:bg-zinc-800"
              to="/sessions"
            >
              {"Sessions"}
              <ArrowUpRight size={13} />
            </Link>
            <Link
              className="inline-flex h-9 min-w-36 items-center justify-between gap-2 rounded-none border border-zinc-200 bg-white px-3 font-medium text-xs transition hover:bg-zinc-50"
              to="/search"
            >
              {"Search archive"}
              <Search size={13} />
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}

export function OverviewHealthStrip({
  activeDays,
  archive,
  facets,
  summary,
}: {
  activeDays: number;
  archive: ArchiveStatus | null;
  facets: Facets | null;
  summary: UsageGlobalTotals | null;
}) {
  return (
    <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      <StatusTile
        detail={`${formatNumber(summary?.projects ?? 0)} projects indexed`}
        icon={Database}
        label={"Dataset"}
        value={`${formatNumber(summary?.sessions ?? 0)} ${"Sessions"}`}
      />
      <StatusTile
        detail={`${formatPlainDate(facets?.dateRange.oldest)} - ${formatPlainDate(facets?.dateRange.newest)}`}
        icon={Clock}
        label={"Usage window"}
        value={`${formatNumber(activeDays)} active days`}
      />
      <StatusTile
        detail={`${formatNumber(archive?.sessionEvents ?? 0)} ${"raw events"}`}
        icon={FileArchive}
        label={"Raw archive"}
        value={formatBytes(archive?.rawBytes ?? 0)}
      />
      <StatusTile
        detail={`${formatNumber(archive?.sessionEvents ?? 0)} events indexed`}
        icon={RefreshCw}
        label={"Latest ingest"}
        value={formatDate(archive?.newestIngestedAt ?? null)}
      />
    </section>
  );
}

export function UsageWindowStrip({ windows }: { windows: UsageWindow[] | null }) {
  return (
    <section className="grid gap-3 xl:grid-cols-3">
      {(windows ?? []).map((window) => (
        <article className="rounded-none border border-zinc-200 bg-white p-4" key={window.label}>
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="font-medium text-xs text-zinc-500 uppercase">{window.label}</div>
              <div className="mt-1 font-semibold text-2xl tabular-nums">
                {formatNumber(window.totals.totalTokens)}
              </div>
            </div>
            <Badge tone="zinc">{formatMoney(window.totals.costUsd)}</Badge>
          </div>
          <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
            <div>
              <div className="text-zinc-400">{"Sessions"}</div>
              <div className="font-medium tabular-nums">{formatNumber(window.totals.sessions)}</div>
            </div>
            <div>
              <div className="text-zinc-400">{"Cache"}</div>
              <div className="font-medium tabular-nums">{formatPercent(window.cacheHitRate)}</div>
            </div>
            <div>
              <div className="text-zinc-400">Patch +</div>
              <div className="font-medium tabular-nums">
                {formatNumber(window.totals.patchAddedLines)}
              </div>
            </div>
          </div>
        </article>
      ))}
    </section>
  );
}

export function StatusTile({
  detail,
  icon: Icon,
  label,
  value,
}: {
  detail: string;
  icon: LucideIcon;
  label: string;
  value: string;
}) {
  return (
    <article className="flex min-w-0 items-center gap-3 rounded-none border border-zinc-200 bg-white p-3">
      <div className="flex size-9 shrink-0 items-center justify-center rounded-none bg-zinc-100 text-zinc-700">
        <Icon size={16} />
      </div>
      <div className="min-w-0">
        <div className="text-zinc-500 text-xs">{label}</div>
        <div className="truncate font-semibold text-sm tabular-nums">{value}</div>
        <div className="truncate text-[11px] text-zinc-400">{detail}</div>
      </div>
    </article>
  );
}

export function HeroStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-none border border-zinc-200 bg-zinc-50/80 px-3 py-2.5">
      <div className="text-zinc-500 text-xs">{label}</div>
      <div className="mt-1 font-semibold text-sm tabular-nums">{value}</div>
    </div>
  );
}
