import { AlertTriangle, Bot, FileJson, HardDrive, Laptop, Layers } from "lucide-react";

import { ArchiveBreakdownList } from "@/components/common/data-display";
import {
  Badge,
  EmptyState,
  ErrorBanner,
  InfoLine,
  MetricCard,
  PageTitle,
  Panel,
  PanelLoading,
} from "@/components/common/primitives";
import { useApiData } from "@/lib/api";
import { formatBytes, formatDate, formatNumber } from "@/lib/format";
import type { ArchiveBreakdown, ArchiveHealth, ArchiveIntegrity, ArchiveStatus } from "@/types";

export function ArchivePage() {
  const archive = useApiData<ArchiveStatus>("/api/archive/status");
  const health = useApiData<ArchiveHealth>("/api/archive/health");
  const byDevice = useApiData<ArchiveBreakdown[]>("/api/archive/devices");
  const byRepository = useApiData<ArchiveBreakdown[]>("/api/archive/repositories?limit=12");
  const integrity = useApiData<ArchiveIntegrity>("/api/archive/integrity");
  const archiveError =
    archive.error || health.error || byDevice.error || byRepository.error || integrity.error;

  return (
    <div className="space-y-6">
      <PageTitle
        eyebrow={"Archive"}
        title={"Raw backup status"}
        description={"Storage status for raw JSONL files and normalized database data."}
      />

      {archiveError ? <ErrorBanner message={archiveError} /> : null}

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <MetricCard
          detail={`${formatNumber(archive.data?.devices ?? 0)} ${"devices"}`}
          icon={Laptop}
          label={"Archived sessions"}
          tone="zinc"
          value={formatNumber(archive.data?.sessions ?? 0)}
        />
        <MetricCard
          detail="jsonl"
          icon={FileJson}
          label={"Raw size"}
          tone="zinc"
          value={formatBytes(archive.data?.rawBytes ?? 0)}
        />
        <MetricCard
          detail={formatDate(archive.data?.oldestSessionTime ?? null)}
          icon={HardDrive}
          label={"Latest ingest"}
          tone="zinc"
          value={formatDate(archive.data?.newestIngestedAt ?? null)}
        />
      </section>

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <MetricCard
          detail={`${formatNumber(archive.data?.sessionEvents ?? 0)} ${"raw events"}`}
          icon={Layers}
          label={"Messages"}
          tone="zinc"
          value={formatNumber(archive.data?.messages ?? 0)}
        />
        <MetricCard
          detail={`${formatNumber(archive.data?.usageEvents ?? 0)} usage rows`}
          icon={Bot}
          label={"Tool events"}
          tone="zinc"
          value={formatNumber(archive.data?.toolEvents ?? 0)}
        />
        <MetricCard
          detail={`${formatNumber(health.data?.verifiedArchiveRows ?? 0)} verified · ${formatNumber(health.data?.missingArchiveRows ?? 0)} missing`}
          icon={AlertTriangle}
          label={"Health"}
          tone="zinc"
          value={health.data?.status === "ok" ? "OK" : "Attention"}
        />
      </section>

      <Panel
        description={"Generated status for normalized, search, and reader data."}
        title={"Archive health"}
      >
        {health.loading ? <PanelLoading /> : null}
        <div className="grid gap-3 bg-zinc-50/50 p-4 md:grid-cols-3">
          <InfoLine
            label="Sessions vs archive rows"
            value={`${formatNumber(health.data?.sessions ?? 0)} / ${formatNumber(health.data?.archiveRows ?? 0)}`}
          />
          <InfoLine
            label={"Conversation turns"}
            value={formatNumber(health.data?.conversationTurns ?? 0)}
          />
          <InfoLine
            label="Search documents"
            value={`${formatNumber(health.data?.defaultSearchDocs ?? 0)} default / ${formatNumber(health.data?.searchDocuments ?? 0)} total`}
          />
          <InfoLine
            label={"Messages / tools"}
            value={`${formatNumber(health.data?.messages ?? 0)} / ${formatNumber(health.data?.toolEvents ?? 0)}`}
          />
          <InfoLine
            label={"Missing raw files"}
            value={formatNumber(health.data?.missingRawFiles ?? 0)}
          />
          <InfoLine
            label={"Verified archives"}
            value={`${formatNumber(health.data?.verifiedArchiveRows ?? 0)} / ${formatNumber(health.data?.archiveRows ?? 0)}`}
          />
          <InfoLine
            label={"Latest ingest"}
            value={formatDate(health.data?.latestIngestedAt ?? null)}
          />
        </div>
      </Panel>

      <Panel description={"Collection and session time range."} title={"Retention window"}>
        {archive.loading ? <PanelLoading /> : null}
        <div className="grid gap-3 bg-zinc-50/50 p-4 md:grid-cols-2">
          <InfoLine
            label={"Oldest session"}
            value={formatDate(archive.data?.oldestSessionTime ?? null)}
          />
          <InfoLine
            label={"Newest session"}
            value={formatDate(archive.data?.newestSessionTime ?? null)}
          />
          <InfoLine
            label={"Oldest ingest"}
            value={formatDate(archive.data?.oldestIngestedAt ?? null)}
          />
          <InfoLine
            label={"Newest ingest"}
            value={formatDate(archive.data?.newestIngestedAt ?? null)}
          />
        </div>
      </Panel>

      <section className="grid gap-6 xl:grid-cols-2">
        <Panel description={"Raw archive volume by Mac/host."} title={"By device"}>
          <ArchiveBreakdownList items={byDevice.data ?? []} loading={byDevice.loading} />
        </Panel>
        <Panel description={"Top raw archive volume by repository."} title={"By repository"}>
          <ArchiveBreakdownList items={byRepository.data ?? []} loading={byRepository.loading} />
        </Panel>
      </section>

      <Panel description={"Raw JSONL file path and metadata status."} title={"Integrity issues"}>
        {integrity.loading ? <PanelLoading /> : null}
        <div className="divide-y divide-zinc-100 bg-white">
          {(integrity.data?.issues ?? []).map((issue) => (
            <div
              className="grid gap-2 px-5 py-3 text-sm md:grid-cols-[240px_1fr_180px]"
              key={`${issue.sessionId}-${issue.problem}`}
            >
              <div className="truncate font-mono text-xs text-zinc-500">{issue.sessionId}</div>
              <div className="truncate text-zinc-600">{issue.path || "-"}</div>
              <Badge tone="zinc">{issue.problem}</Badge>
            </div>
          ))}
        </div>
        {!integrity.loading && (integrity.data?.issues ?? []).length === 0 ? (
          <EmptyState label={"No integrity issues to show"} />
        ) : null}
      </Panel>
    </div>
  );
}
