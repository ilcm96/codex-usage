import type { LucideIcon } from "lucide-react";
import { AlertTriangle, FolderGit2, Loader2 } from "lucide-react";
import type React from "react";

import { formatNumber } from "@/lib/format";

export function PageTitle({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3">
      <div>
        <div className="font-medium text-[11px] text-zinc-500 uppercase tracking-wide">
          {eyebrow}
        </div>
        <h2 className="mt-1 font-semibold text-2xl tracking-normal">{title}</h2>
        <p className="mt-1 text-sm text-zinc-500">{description}</p>
      </div>
      <div className="hidden items-center gap-2 text-zinc-400 text-xs md:flex">
        <FolderGit2 size={14} />
        repo + cwd indexed
      </div>
    </div>
  );
}

export function MetricCard({
  label,
  value,
  detail,
  icon: Icon,
  tone,
}: {
  label: string;
  value: string;
  detail: string;
  icon: LucideIcon;
  tone: "zinc";
}) {
  const tones = {
    zinc: "bg-zinc-100 text-zinc-700 ring-zinc-200",
  };

  return (
    <article className="rounded-none border border-zinc-200 bg-white p-4">
      <div className="flex items-center justify-between gap-3">
        <div className="text-sm text-zinc-500">{label}</div>
        <div
          className={`flex size-8 items-center justify-center rounded-none ring-1 ${tones[tone]}`}
        >
          <Icon size={16} />
        </div>
      </div>
      <div className="mt-3 break-words font-semibold text-2xl tabular-nums">{value}</div>
      <div className="mt-1 text-xs text-zinc-500">{detail}</div>
    </article>
  );
}

export function Panel({
  title,
  description,
  action,
  children,
}: {
  title: string;
  description: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="min-w-0 overflow-visible rounded-none border border-zinc-200 bg-white">
      <div className="flex flex-wrap items-center justify-between gap-3 border-zinc-200 border-b px-5 py-4">
        <div>
          <h3 className="font-semibold text-base">{title}</h3>
          <p className="text-xs text-zinc-500">{description}</p>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

export function UsageRow({
  detail,
  label,
  value,
  max,
}: {
  detail?: string;
  label: string;
  value: number;
  max: number;
}) {
  const width = Math.max(3, Math.round((value / max) * 100));

  return (
    <div>
      <div className="mb-1 flex items-center justify-between gap-3 text-sm">
        <span className="truncate font-medium">{label}</span>
        <span className="shrink-0 tabular-nums text-zinc-500 text-xs">{formatNumber(value)}</span>
      </div>
      {detail ? <div className="mb-1 truncate text-[11px] text-zinc-400">{detail}</div> : null}
      <div className="h-2 rounded-none bg-zinc-100">
        <div className="h-2 rounded-none bg-zinc-950" style={{ width: `${width}%` }} />
      </div>
    </div>
  );
}

export function InfoLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-none border border-zinc-200 bg-white px-3 py-2.5">
      <div className="text-zinc-500 text-xs">{label}</div>
      <div className="mt-1 font-medium text-sm">{value}</div>
    </div>
  );
}

export function Badge({ children, tone }: { children: React.ReactNode; tone: "zinc" }) {
  const tones = {
    zinc: "border-zinc-200 bg-zinc-50 text-zinc-600",
  };

  return (
    <span
      className={`inline-flex h-6 w-fit shrink-0 items-center gap-1 rounded-none border px-2 font-medium text-xs ${tones[tone]}`}
    >
      {children}
    </span>
  );
}

export function ChartLegend({
  active = true,
  color,
  label,
  onClick,
  value,
}: {
  active?: boolean;
  color: string;
  label: string;
  onClick?: () => void;
  value: string;
}) {
  const className = `flex items-center gap-2 rounded-none border border-zinc-200 bg-zinc-50 px-2.5 py-1.5 transition ${
    active ? "opacity-100" : "opacity-45"
  } ${onClick ? "cursor-pointer hover:border-zinc-300 hover:bg-white" : ""}`;
  const content = (
    <>
      <span className={`size-2 rounded-none ${color}`} />
      <span className="text-zinc-500 text-xs">{label}</span>
      <span className="font-medium text-xs tabular-nums">{value}</span>
    </>
  );

  if (!onClick) {
    return <div className={className}>{content}</div>;
  }

  return (
    <button aria-pressed={active} className={className} onClick={onClick} type="button">
      {content}
    </button>
  );
}

export function ErrorBanner({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 rounded-none border border-zinc-200 bg-zinc-50 px-4 py-3 text-sm text-zinc-800">
      <AlertTriangle className="mt-0.5 shrink-0" size={16} />
      <div>
        <div className="font-medium">Some data could not be loaded.</div>
        <div className="mt-0.5 text-xs text-zinc-600">{message}</div>
      </div>
    </div>
  );
}

export function PanelError({ message }: { message: string }) {
  return (
    <div className="flex items-center gap-2 p-4 text-sm text-zinc-600">
      <AlertTriangle size={16} />
      {message}
    </div>
  );
}

export function PanelLoading({ label = "Loading" }: { label?: string }) {
  return (
    <div className="flex items-center gap-2 p-4 text-sm text-zinc-500">
      <Loader2 className="animate-spin" size={16} />
      {label === "Loading" ? "Loading" : label}
    </div>
  );
}

export function EmptyState({ label }: { label: string }) {
  return <div className="p-6 text-center text-sm text-zinc-500">{label}</div>;
}

export function LoadingScreen() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-50 text-zinc-500">
      <Loader2 className="animate-spin" size={22} />
    </main>
  );
}
