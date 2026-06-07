import { Download } from "lucide-react";
import { type FocusEvent, type MouseEvent, useState } from "react";

import { EmptyState, PanelLoading, UsageRow } from "@/components/common/primitives";
import { apiBase } from "@/lib/api";
import { formatBytes, formatMoney, formatNumber, formatPlainDate } from "@/lib/format";
import type { ArchiveBreakdown, UsageCalendarDay } from "@/types";

export function CalendarHeatmap({
  items,
  loading,
}: {
  items: UsageCalendarDay[];
  loading: boolean;
}) {
  const itemByDate = new Map(items.map((item) => [item.date, item]));
  const days = buildCalendarDays(itemByDate);
  const weeks = chunkWeeks(days);
  const maxTokens = Math.max(
    ...days.flatMap((cell) => (cell.kind === "day" ? [cell.item.totalTokens] : [])),
    1,
  );
  const monthLabels = buildMonthLabels(weeks);
  const [hovered, setHovered] = useState<CalendarTooltip | null>(null);

  return (
    <div className="p-4">
      {loading ? <PanelLoading /> : null}
      <div className="overflow-x-auto pb-2 [scrollbar-color:#a1a1aa_#f4f4f5] [scrollbar-width:thin]">
        <div className="min-w-[900px] md:min-w-0">
          <div className="mb-1 grid grid-cols-[28px_1fr] gap-2">
            <div />
            <div
              className="grid overflow-hidden text-[10px] text-zinc-400"
              style={{ gridTemplateColumns: `repeat(${weeks.length}, minmax(0, 1fr))` }}
            >
              {monthLabels.map((label) => (
                <div className="h-4 overflow-visible whitespace-nowrap" key={label.key}>
                  {label.value}
                </div>
              ))}
            </div>
          </div>
          <div className="grid grid-cols-[28px_1fr] gap-2">
            <div className="grid grid-rows-7 gap-1 text-[10px] text-zinc-400">
              <div />
              <div>Mon</div>
              <div />
              <div>Wed</div>
              <div />
              <div>Fri</div>
              <div />
            </div>
            <div
              className="grid grid-flow-col grid-rows-7 gap-1"
              style={{ gridTemplateColumns: `repeat(${weeks.length}, minmax(0, 1fr))` }}
            >
              {weeks.flat().map((cell) => {
                if (cell.kind === "empty") {
                  return <div className="aspect-square w-full" key={cell.key} />;
                }

                const item = cell.item;
                const level = calendarLevel(item.totalTokens, maxTokens);
                const colors = [
                  "bg-zinc-200 hover:bg-zinc-300",
                  "bg-zinc-300 hover:bg-zinc-400",
                  "bg-zinc-500 hover:bg-zinc-600",
                  "bg-zinc-700 hover:bg-zinc-800",
                  "bg-zinc-950 hover:bg-black",
                ];
                const tooltip = `${formatPlainDate(item.date)} · ${formatNumber(item.totalTokens)} tokens · ${formatMoney(item.costUsd)} · ${formatNumber(item.projects)} projects`;

                return (
                  <button
                    aria-label={tooltip}
                    className={`block aspect-square w-full rounded-[2px] border border-white p-0 focus:outline-none focus:ring-1 focus:ring-zinc-950 ${colors[level] ?? colors[0]}`}
                    key={item.date}
                    onBlur={() => setHovered(null)}
                    onFocus={(event) => setHovered(calendarTooltipFromFocus(event, tooltip))}
                    onMouseEnter={(event) => setHovered(calendarTooltipFromMouse(event, tooltip))}
                    onMouseLeave={() => setHovered(null)}
                    onMouseMove={(event) => setHovered(calendarTooltipFromMouse(event, tooltip))}
                    type="button"
                  />
                );
              })}
            </div>
          </div>
        </div>
      </div>
      {hovered ? (
        <div
          className="pointer-events-none fixed z-50 whitespace-nowrap border border-zinc-800 bg-zinc-950 px-2 py-1 font-medium text-[11px] text-white shadow-sm"
          style={{
            left: hovered.x,
            top: hovered.y,
            transform: tooltipTransform(hovered),
          }}
        >
          {hovered.text}
        </div>
      ) : null}
      {!loading && items.length === 0 ? <EmptyState label={"No calendar data"} /> : null}
    </div>
  );
}

type CalendarTooltip = {
  horizontal: "center" | "left" | "right";
  text: string;
  vertical: "above" | "below";
  x: number;
  y: number;
};

function calendarTooltipFromMouse(
  event: MouseEvent<HTMLButtonElement>,
  text: string,
): CalendarTooltip {
  return buildCalendarTooltip(event.clientX, event.clientY, text);
}

function calendarTooltipFromFocus(
  event: FocusEvent<HTMLButtonElement>,
  text: string,
): CalendarTooltip {
  const rect = event.currentTarget.getBoundingClientRect();
  return buildCalendarTooltip(rect.left + rect.width / 2, rect.top, text);
}

function buildCalendarTooltip(x: number, y: number, text: string): CalendarTooltip {
  const horizontal = x < 180 ? "left" : x > window.innerWidth - 260 ? "right" : "center";
  const vertical = y < 80 ? "below" : "above";

  return {
    horizontal,
    text,
    vertical,
    x,
    y: vertical === "above" ? y - 10 : y + 18,
  };
}

function tooltipTransform(tooltip: CalendarTooltip) {
  const horizontal = {
    center: "translateX(-50%)",
    left: "translateX(0)",
    right: "translateX(-100%)",
  }[tooltip.horizontal];
  const vertical = tooltip.vertical === "above" ? "translateY(-100%)" : "translateY(0)";

  return `${horizontal} ${vertical}`;
}

type CalendarCell =
  | {
      item: UsageCalendarDay;
      kind: "day";
    }
  | {
      key: string;
      kind: "empty";
    };

function buildCalendarDays(itemByDate: Map<string, UsageCalendarDay>) {
  const today = startOfDay(new Date());
  const start = addDays(today, -364);
  const firstGridDate = addDays(start, -start.getDay());
  const out: CalendarCell[] = [];

  for (let date = firstGridDate; date <= today; date = addDays(date, 1)) {
    const key = toDateKey(date);
    if (date < start) {
      out.push({ key: `empty-${key}`, kind: "empty" });
      continue;
    }

    out.push({
      item: itemByDate.get(key) ?? {
        costUsd: 0,
        date: key,
        projects: 0,
        totalTokens: 0,
      },
      kind: "day",
    });
  }

  return out;
}

function chunkWeeks(days: CalendarCell[]) {
  const weeks: CalendarCell[][] = [];
  for (let index = 0; index < days.length; index += 7) {
    weeks.push(days.slice(index, index + 7));
  }
  return weeks;
}

function buildMonthLabels(weeks: CalendarCell[][]) {
  return weeks.map((week, index) => {
    const firstDay = week.find((cell) => cell.kind === "day")?.item;
    const previousWeek = weeks[index - 1];
    const previousDay = previousWeek?.find((cell) => cell.kind === "day")?.item;
    const shouldShow =
      firstDay &&
      (!previousDay ||
        new Date(`${firstDay.date}T00:00:00`).getMonth() !==
          new Date(`${previousDay.date}T00:00:00`).getMonth());

    return {
      key: `month-${index}-${firstDay?.date ?? "empty"}`,
      value: shouldShow
        ? new Intl.DateTimeFormat("en-US", { month: "short" }).format(
            new Date(`${firstDay.date}T00:00:00`),
          )
        : "",
    };
  });
}

function calendarLevel(tokens: number, maxTokens: number) {
  if (tokens <= 0) {
    return 0;
  }
  return Math.max(1, Math.ceil((tokens / maxTokens) * 4));
}

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function addDays(date: Date, days: number) {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

function toDateKey(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function HighlightedText({
  end,
  start,
  text,
}: {
  end: number;
  start: number;
  text: string;
}) {
  if (start < 0 || end <= start || start >= text.length) {
    return <>{text}</>;
  }

  return (
    <>
      {text.slice(0, start)}
      <mark className="bg-zinc-200 px-0.5 text-zinc-950">{text.slice(start, end)}</mark>
      {text.slice(end)}
    </>
  );
}

export function BulkExportLink({ format, query }: { format: string; query: string }) {
  const params = new URLSearchParams(query);
  params.set("format", format);

  return (
    <a
      className="inline-flex h-10 items-center justify-center gap-1.5 rounded-none border border-zinc-200 bg-white px-3 font-medium text-sm transition hover:bg-zinc-50"
      href={`${apiBase}/api/exports/sessions?${params.toString()}`}
    >
      <Download size={14} />
      {format}
    </a>
  );
}

export function ArchiveBreakdownList({
  items,
  loading,
}: {
  items: ArchiveBreakdown[];
  loading: boolean;
}) {
  const maxBytes = Math.max(...items.map((item) => item.rawBytes), 1);

  return (
    <div className="space-y-4 p-4">
      {items.map((item) => (
        <UsageRow
          detail={`${formatNumber(item.sessions)} sessions · ${formatBytes(item.rawBytes)} raw · ${item.hostname || item.url || ""}`}
          key={item.id || item.name}
          label={item.name || "local"}
          max={maxBytes}
          value={item.rawBytes}
        />
      ))}
      {loading ? <PanelLoading /> : null}
      {!loading && items.length === 0 ? <EmptyState label={"No archive data"} /> : null}
    </div>
  );
}
