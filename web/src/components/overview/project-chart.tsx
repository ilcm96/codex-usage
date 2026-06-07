import { firstNonEmpty, formatNumber, pathTail, truncateMiddle } from "@/lib/format";
import type { Project, ProjectChartItem } from "@/types";

export function buildProjectChartData(projects: Project[], limit: number): ProjectChartItem[] {
  const topProjects = projects.slice(0, limit);
  const labelCounts = new Map<string, number>();
  for (const project of topProjects) {
    const label = project.displayName || pathTail(project.cwd) || "unknown";
    labelCounts.set(label, (labelCounts.get(label) ?? 0) + 1);
  }

  const seenLabels = new Map<string, number>();
  return topProjects.map((project) => {
    const baseLabel = project.displayName || pathTail(project.cwd) || "unknown";
    const seen = (seenLabels.get(baseLabel) ?? 0) + 1;
    seenLabels.set(baseLabel, seen);

    const duplicate = (labelCounts.get(baseLabel) ?? 0) > 1;
    const detail = firstNonEmpty(project.relativePath, project.repository, project.cwd);
    const suffix = duplicate ? ` · ${truncateMiddle(firstNonEmpty(detail, project.id), 16)}` : "";

    return {
      ...project,
      chartDetail: detail,
      chartKey: `${project.id}:${project.cwd}:${project.repository}:${seen}`,
      chartLabel: `${baseLabel}${suffix}`,
    };
  });
}

export function ProjectRankingTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: Array<{ payload?: ProjectChartItem; value?: number }>;
}) {
  const item = payload?.[0]?.payload;
  if (!active || !item) {
    return null;
  }

  return (
    <div className="border border-zinc-200 bg-white px-3 py-2 text-xs">
      <div className="font-medium text-zinc-950">{item.displayName || "unknown"}</div>
      <div className="mt-1 max-w-[280px] truncate text-zinc-500">{item.chartDetail}</div>
      <div className="mt-2 font-semibold tabular-nums">{formatNumber(item.totalTokens)} tokens</div>
    </div>
  );
}
