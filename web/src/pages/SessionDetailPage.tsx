import {
  BarChart3,
  ChevronLeft,
  ChevronRight,
  CircleDollarSign,
  FilePlus2,
  MessageSquareText,
} from "lucide-react";
import { useState } from "react";
import { useParams } from "react-router-dom";

import {
  EmptyState,
  InfoLine,
  MetricCard,
  PageTitle,
  Panel,
  PanelError,
  PanelLoading,
  UsageRow,
} from "@/components/common/primitives";
import { ExportLinks, ReaderTurnCard, TimelineCard } from "@/components/sessions/session-cards";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { useApiData } from "@/lib/api";
import { fieldClassName } from "@/lib/constants";
import {
  formatDate,
  formatDuration,
  formatMoney,
  formatNumber,
  formatPercent,
  sessionDisplayTitle,
  truncateMiddle,
} from "@/lib/format";
import type { ReaderResponse, SessionDetail, TimelineResponse } from "@/types";

export function SessionDetailPage() {
  const { sessionId = "" } = useParams();
  const detail = useApiData<SessionDetail>(
    sessionId ? `/api/sessions/${encodeURIComponent(sessionId)}` : "",
  );
  const [activeTab, setActiveTab] = useState("reader");
  const [readerOffset, setReaderOffset] = useState(0);
  const [readerQuery, setReaderQuery] = useState("");
  const [timelineOffset, setTimelineOffset] = useState(0);
  const [timelineQuery, setTimelineQuery] = useState("");
  const [timelineKind, setTimelineKind] = useState("");
  const session = detail.data?.session;
  const models = detail.data?.models ?? [];
  const readerParams = new URLSearchParams();
  readerParams.set("limit", "30");
  readerParams.set("offset", String(readerOffset));
  if (readerQuery.trim()) {
    readerParams.set("q", readerQuery.trim());
  }
  const reader = useApiData<ReaderResponse>(
    sessionId ? `/api/sessions/${encodeURIComponent(sessionId)}/reader?${readerParams}` : "",
  );
  const readerItems = reader.data?.items ?? [];
  const timelineParams = new URLSearchParams();
  timelineParams.set("limit", "100");
  timelineParams.set("offset", String(timelineOffset));
  if (timelineQuery.trim()) {
    timelineParams.set("q", timelineQuery.trim());
  }
  if (timelineKind) {
    timelineParams.set("kind", timelineKind);
  }
  const timeline = useApiData<TimelineResponse>(
    sessionId ? `/api/sessions/${encodeURIComponent(sessionId)}/timeline?${timelineParams}` : "",
  );
  const timelineItems = timeline.data?.items ?? [];

  return (
    <div className="space-y-6">
      <PageTitle
        eyebrow={"Session detail"}
        title={
          reader.data?.summary.displayTitle ||
          session?.displayTitle ||
          reader.data?.summary.title ||
          (session ? sessionDisplayTitle(session) : "") ||
          session?.project ||
          session?.repository ||
          "Conversation detail"
        }
        description={
          reader.data?.summary.userIntent ||
          session?.userIntent ||
          reader.data?.summary.shortSummary ||
          session?.shortSummary ||
          session?.cwd ||
          sessionId
        }
      />

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          detail={`${formatNumber(session?.inputTokens ?? 0)} ${"Input"} · ${formatNumber(session?.cachedInputTokens ?? 0)} cached`}
          icon={BarChart3}
          label={"Total tokens"}
          tone="zinc"
          value={formatNumber(session?.totalTokens ?? 0)}
        />
        <MetricCard
          detail={`${formatNumber(session?.outputTokens ?? 0)} ${"Output"} · ${formatNumber(session?.reasoningOutputTokens ?? 0)} reasoning`}
          icon={CircleDollarSign}
          label={"Cost"}
          tone="zinc"
          value={formatMoney(session?.costUsd ?? 0)}
        />
        <MetricCard
          detail={`${formatNumber(reader.data?.summary.userMessageCount ?? session?.userMessageCount ?? 0)} ${"User"} · ${formatNumber(reader.data?.summary.assistantMessageCount ?? session?.assistantMessageCount ?? 0)} ${"Assistant"}`}
          icon={MessageSquareText}
          label={"Messages"}
          tone="zinc"
          value={formatNumber(session?.messageCount ?? 0)}
        />
        <MetricCard
          detail={`${formatNumber(session?.toolCallCount ?? 0)} ${"Tool calls"}`}
          icon={FilePlus2}
          label={"Patch added lines"}
          tone="zinc"
          value={formatNumber(session?.patchAddedLines ?? 0)}
        />
      </section>

      <Panel
        action={session ? <ExportLinks sessionId={session.id} /> : null}
        description={`${formatNumber(reader.data?.total ?? 0)} reader turns · ${formatNumber(timeline.data?.total ?? 0)} ${"raw events"}`}
        title={"Conversation"}
      >
        <div className="flex flex-wrap gap-1 border-zinc-200 border-b bg-zinc-50/70 p-3">
          {["reader", "timeline", "usage", "raw"].map((tab) => (
            <button
              className={`h-8 rounded-none border px-3 font-medium text-xs uppercase ${
                activeTab === tab
                  ? "border-zinc-950 bg-zinc-950 text-white"
                  : "border-zinc-200 bg-white text-zinc-600 hover:bg-zinc-50"
              }`}
              key={tab}
              onClick={() => setActiveTab(tab)}
              type="button"
            >
              {tab}
            </button>
          ))}
        </div>

        {activeTab === "reader" ? (
          <>
            <div className="grid gap-2 border-zinc-200 border-b bg-zinc-50/70 p-3 md:grid-cols-[minmax(0,1fr)_220px]">
              <input
                className={fieldClassName}
                onChange={(event) => {
                  setReaderQuery(event.target.value);
                  setReaderOffset(0);
                }}
                placeholder={"Search readable conversation"}
                value={readerQuery}
              />
              <div className="flex gap-2">
                <Button
                  className="h-10 flex-1"
                  disabled={readerOffset <= 0}
                  onClick={() => setReaderOffset(Math.max(0, readerOffset - 30))}
                  type="button"
                  variant="outline"
                >
                  <ChevronLeft size={14} />
                  {"Prev"}
                </Button>
                <Button
                  className="h-10 flex-1"
                  disabled={!reader.data?.nextOffset}
                  onClick={() => setReaderOffset(reader.data?.nextOffset ?? 0)}
                  type="button"
                  variant="outline"
                >
                  {"Next"}
                  <ChevronRight size={14} />
                </Button>
              </div>
            </div>
            {reader.loading || detail.loading ? <PanelLoading label={"Loading reader..."} /> : null}
            {!reader.loading && reader.error ? <PanelError message={reader.error} /> : null}
            <div className="space-y-4 bg-zinc-50/50 p-4">
              {readerItems.map((item) => (
                <ReaderTurnCard item={item} key={item.turnIndex} />
              ))}
              {!reader.loading && readerItems.length === 0 ? (
                <EmptyState label={"No readable conversation turns"} />
              ) : null}
            </div>
          </>
        ) : null}

        {activeTab === "timeline" ? (
          <>
            <div className="grid gap-2 border-zinc-200 border-b bg-zinc-50/70 p-3 md:grid-cols-[minmax(0,1fr)_160px_220px]">
              <input
                className={fieldClassName}
                onChange={(event) => {
                  setTimelineQuery(event.target.value);
                  setTimelineOffset(0);
                }}
                placeholder={"Search raw event timeline"}
                value={timelineQuery}
              />
              <Select
                onValueChange={(value) => {
                  setTimelineKind(value);
                  setTimelineOffset(0);
                }}
                options={[
                  { label: "All events", value: "" },
                  { label: "Messages", value: "message" },
                  { label: "Tools", value: "tool" },
                ]}
                placeholder={"Event kind"}
                value={timelineKind}
              />
              <div className="flex gap-2">
                <Button
                  className="h-10 flex-1"
                  disabled={timelineOffset <= 0}
                  onClick={() => setTimelineOffset(Math.max(0, timelineOffset - 100))}
                  type="button"
                  variant="outline"
                >
                  <ChevronLeft size={14} />
                  {"Prev"}
                </Button>
                <Button
                  className="h-10 flex-1"
                  disabled={!timeline.data?.nextOffset}
                  onClick={() => setTimelineOffset(timeline.data?.nextOffset ?? 0)}
                  type="button"
                  variant="outline"
                >
                  {"Next"}
                  <ChevronRight size={14} />
                </Button>
              </div>
            </div>
            {timeline.loading || detail.loading ? <PanelLoading /> : null}
            <div className="max-h-[720px] overflow-auto bg-zinc-50/50 p-4">
              {timelineItems.map((item) => (
                <TimelineCard item={item} key={`${item.kind}-${item.seq}`} />
              ))}
              {!timeline.loading && timelineItems.length === 0 ? (
                <EmptyState label={"No conversation content"} />
              ) : null}
            </div>
          </>
        ) : null}

        {activeTab === "usage" ? (
          <div className="space-y-4 p-4">
            {models.map((item) => (
              <UsageRow
                detail={`${formatMoney(item.costUsd)} · ${formatNumber(item.inputTokens)} ${"Input"} · ${formatNumber(item.outputTokens + item.reasoningOutputTokens)} ${"Output"}`}
                key={`usage-tab-${item.model}`}
                label={item.model || "unknown"}
                max={Math.max(...models.map((model) => model.totalTokens), 1)}
                value={item.totalTokens}
              />
            ))}
            {!detail.loading && models.length === 0 ? (
              <EmptyState label={"No model usage"} />
            ) : null}
          </div>
        ) : null}

        {activeTab === "raw" ? (
          <div className="grid gap-3 bg-zinc-50/50 p-4 md:grid-cols-2">
            <InfoLine label="Session ID" value={sessionId} />
            <InfoLine
              label="Search documents"
              value={formatNumber(reader.data?.summary.searchableDocumentRows ?? 0)}
            />
            <InfoLine
              label={"Conversation turns"}
              value={formatNumber(reader.data?.summary.conversationTurnCount ?? 0)}
            />
            <InfoLine
              label={"Duration"}
              value={formatDuration(
                reader.data?.summary.durationSeconds ?? session?.durationSeconds ?? 0,
              )}
            />
          </div>
        ) : null}
      </Panel>

      <details className="rounded-none border border-zinc-200 bg-white">
        <summary className="cursor-pointer border-zinc-200 border-b px-5 py-4 font-semibold text-base">
          {"Session metadata and model usage"}
        </summary>
        <section className="grid gap-6 p-4 xl:grid-cols-[minmax(0,1fr)_420px]">
          <div>
            <div className="mb-3 text-xs text-zinc-500">{"Session metadata"}</div>
            <div className="grid gap-3 md:grid-cols-2">
              <InfoLine label={"Repository"} value={session?.repository || "local"} />
              <InfoLine label={"Branch"} value={session?.branch || "-"} />
              <InfoLine
                label="Commit"
                value={session?.commitHash ? truncateMiddle(session.commitHash, 24) : "-"}
              />
              <InfoLine label={"Updated"} value={formatDate(session?.updatedAt ?? null)} />
              <InfoLine label={"Cache hit"} value={formatPercent(session?.cacheHitRate ?? 0)} />
              <InfoLine
                label={"Patch added lines"}
                value={formatNumber(session?.patchAddedLines ?? 0)}
              />
              <InfoLine
                label="Indexed docs"
                value={`${formatNumber(session?.searchableMessages ?? 0)} ${"Messages"} / ${formatNumber(session?.searchableTools ?? 0)} ${"Tools"}`}
              />
            </div>
          </div>
          <div>
            <div className="mb-3 text-xs text-zinc-500">{"Usage analytics"}</div>
            <div className="space-y-4">
              {models.map((item) => (
                <UsageRow
                  detail={formatMoney(item.costUsd)}
                  key={item.model}
                  label={item.model || "unknown"}
                  max={Math.max(...models.map((model) => model.totalTokens), 1)}
                  value={item.totalTokens}
                />
              ))}
              {detail.loading ? <PanelLoading /> : null}
              {!detail.loading && models.length === 0 ? (
                <EmptyState label={"No model usage"} />
              ) : null}
            </div>
          </div>
        </section>
      </details>
    </div>
  );
}
