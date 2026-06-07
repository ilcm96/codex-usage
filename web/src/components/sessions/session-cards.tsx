import { Database, FileJson, FileText } from "lucide-react";
import { Link } from "react-router-dom";

import { Badge } from "@/components/common/primitives";
import { apiBase } from "@/lib/api";
import { formatDate, formatNumber, sessionDisplaySummary, sessionDisplayTitle } from "@/lib/format";
import type { ReaderTurn, Session, TimelineItem } from "@/types";

export function SessionListItem({ session }: { session: Session }) {
  return (
    <Link
      className="block px-5 py-3.5 text-sm transition hover:bg-zinc-50"
      to={`/sessions/${encodeURIComponent(session.id)}`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium">{sessionDisplayTitle(session)}</div>
          <div className="mt-1 flex flex-wrap gap-2 text-xs text-zinc-500">
            <Badge tone="zinc">{session.repository || "local"}</Badge>
            <span>{session.project || "unknown"}</span>
          </div>
          {sessionDisplaySummary(session) ? (
            <div className="mt-2 line-clamp-2 text-xs text-zinc-500">
              {sessionDisplaySummary(session)}
            </div>
          ) : null}
        </div>
        <div className="text-right text-xs text-zinc-500">
          <div>{formatDate(session.updatedAt ?? session.startedAt)}</div>
          <div className="mt-1 tabular-nums">{formatNumber(session.totalTokens)}</div>
          <div className="mt-1 tabular-nums">+{formatNumber(session.patchAddedLines ?? 0)}</div>
        </div>
      </div>
    </Link>
  );
}

export function ReaderTurnCard({ item }: { item: ReaderTurn }) {
  return (
    <article className="rounded-none border border-zinc-200 bg-white">
      <div className="flex flex-wrap items-center justify-between gap-2 border-zinc-200 border-b px-4 py-2 text-xs">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="zinc">
            {"turns"} #{item.turnIndex + 1}
          </Badge>
          {item.toolCallCount || item.toolResultCount ? (
            <span className="text-zinc-500">
              {formatNumber(item.toolCallCount)} calls · {formatNumber(item.toolResultCount)}{" "}
              results
            </span>
          ) : null}
          {item.toolNames.slice(0, 4).map((name) => (
            <span className="border border-zinc-200 px-1.5 py-0.5 text-zinc-500" key={name}>
              {name}
            </span>
          ))}
        </div>
        <span className="text-zinc-400">{formatDate(item.startedAt)}</span>
      </div>
      <div className="grid gap-0 md:grid-cols-2">
        <div className="border-zinc-200 border-b p-4 md:border-r md:border-b-0">
          <div className="mb-2 font-medium text-xs text-zinc-500 uppercase">{"User"}</div>
          <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap break-words text-sm leading-6 text-zinc-800">
            {item.userText || "-"}
          </pre>
        </div>
        <div className="p-4">
          <div className="mb-2 font-medium text-xs text-zinc-500 uppercase">{"Assistant"}</div>
          <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap break-words text-sm leading-6 text-zinc-800">
            {item.assistantText || "-"}
          </pre>
        </div>
      </div>
    </article>
  );
}

export function TimelineCard({ item }: { item: TimelineItem }) {
  return (
    <article className="mb-3 rounded-none border border-zinc-200 bg-white p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2 text-xs">
        <span className="flex flex-wrap items-center gap-1.5 font-medium text-zinc-700">
          <Badge tone="zinc">{item.kind}</Badge>
          {item.role ? <span>{item.role}</span> : null}
          {item.toolName ? <span>{item.toolName}</span> : null}
          {item.status ? <span>{item.status}</span> : null}
        </span>
        <span className="text-zinc-400">#{item.seq}</span>
      </div>
      <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs leading-relaxed text-zinc-700">
        {item.text}
      </pre>
    </article>
  );
}

export function ExportLinks({ sessionId }: { sessionId: string }) {
  return (
    <div className="flex flex-wrap gap-2">
      {[
        ["raw", FileJson],
        ["json", Database],
        ["markdown", FileText],
      ].map(([format, Icon]) => (
        <a
          className="inline-flex h-8 items-center gap-1.5 rounded-none border border-zinc-200 bg-white px-2.5 font-medium text-xs transition hover:bg-zinc-50"
          href={`${apiBase}/api/sessions/${encodeURIComponent(sessionId)}/export?format=${format}`}
          key={format as string}
        >
          <Icon size={13} />
          {format as string}
        </a>
      ))}
    </div>
  );
}
