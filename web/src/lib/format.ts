import type { FacetOption, Session } from "@/types";

export function firstNonEmpty(...values: string[]) {
  return values.find((value) => value.trim() !== "") ?? "";
}

export function sessionDisplayTitle(session: Partial<Session>) {
  return firstNonEmpty(
    session.displayTitle ?? "",
    session.title ?? "",
    session.userIntent ?? "",
    session.project ?? "",
    session.repository ?? "",
    pathTail(session.cwd ?? ""),
    "Untitled session",
  );
}

export function sessionDisplaySummary(session: Partial<Session>) {
  return firstNonEmpty(
    session.userIntent ?? "",
    session.shortSummary ?? "",
    session.firstUserMessage ?? "",
    session.lastUserMessage ?? "",
  );
}

export function pathTail(value: string) {
  const parts = value.split("/").filter(Boolean);
  return parts.at(-1) ?? "";
}

export function formatFacetLabel(option: FacetOption, duplicate: boolean) {
  if (!duplicate || !option.detail) {
    return option.label;
  }
  return `${option.label} · ${truncateMiddle(option.detail, 32)}`;
}

export function formatNumber(value: number) {
  if (Math.abs(value) >= 1_000_000) {
    return formatMillionBillion(value);
  }

  return new Intl.NumberFormat("en-US", {
    notation: "standard",
  }).format(value);
}

export function formatCompact(value: number) {
  if (Math.abs(value) >= 1_000_000) {
    return formatMillionBillion(value);
  }

  return new Intl.NumberFormat("en-US", { notation: "compact" }).format(value);
}

export function formatMoney(value: number) {
  return new Intl.NumberFormat("en-US", { currency: "USD", style: "currency" }).format(value);
}

export function formatPercent(value: number) {
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 1,
    style: "percent",
  }).format(value);
}

export function formatDate(value: string | null) {
  if (!value) {
    return "-";
  }
  return new Intl.DateTimeFormat("en-US", {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(new Date(value));
}

export function formatPlainDate(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Intl.DateTimeFormat("en-US", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(new Date(`${value}T00:00:00`));
}

export function formatShortDate(value: string) {
  return new Intl.DateTimeFormat("en-US", {
    day: "2-digit",
    month: "2-digit",
  }).format(new Date(`${value}T00:00:00`));
}

export function formatBytes(value: number) {
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 1,
    notation: "compact",
    style: "unit",
    unit: "byte",
    unitDisplay: "narrow",
  }).format(value);
}

export function formatDuration(seconds: number) {
  if (!seconds || seconds < 0) {
    return "-";
  }
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${Math.max(1, minutes)}m`;
}

export function truncateMiddle(value: string, maxLength: number) {
  if (value.length <= maxLength) {
    return value;
  }
  const edge = Math.max(3, Math.floor((maxLength - 3) / 2));
  return `${value.slice(0, edge)}...${value.slice(-edge)}`;
}

function formatMillionBillion(value: number) {
  const abs = Math.abs(value);
  const divisor = abs >= 1_000_000_000 ? 1_000_000_000 : 1_000_000;
  const suffix = abs >= 1_000_000_000 ? "B" : "M";
  const scaled = value / divisor;
  const formatted = new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 1,
    minimumFractionDigits: Number.isInteger(scaled) ? 0 : undefined,
  }).format(scaled);

  return `${formatted}${suffix}`;
}
