import { ChevronDown, ChevronUp, Filter } from "lucide-react";
import { useState } from "react";

import { Badge, Panel } from "@/components/common/primitives";
import { Button } from "@/components/ui/button";
import { DatePicker } from "@/components/ui/date-picker";
import { Select, type SelectOption } from "@/components/ui/select";
import { emptyDashboardFilters } from "@/lib/constants";
import { formatFacetLabel } from "@/lib/format";
import type { DashboardFilters, FacetOption, Facets } from "@/types";

export function DashboardFilterBar({
  bucket,
  facets,
  filters,
  onBucketChange,
  onFiltersChange,
  showBucket = false,
}: {
  bucket?: string;
  facets: Facets | null;
  filters: DashboardFilters;
  onBucketChange?: (bucket: string) => void;
  onFiltersChange: (filters: DashboardFilters) => void;
  showBucket?: boolean;
}) {
  const [expanded, setExpanded] = useState(false);
  const activeFilters = Object.values(filters).filter(Boolean).length;

  function update(key: keyof DashboardFilters, value: string) {
    onFiltersChange({ ...filters, [key]: value });
  }

  return (
    <Panel
      action={
        <Button
          onClick={() => setExpanded((value) => !value)}
          size="sm"
          type="button"
          variant="outline"
        >
          <Filter size={14} />
          {activeFilters ? `${activeFilters} active` : "Filters"}
          {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </Button>
      }
      description={"Narrow the whole screen by date, repo, cwd, model, and device."}
      title={"Filters"}
    >
      <div className="flex flex-wrap gap-2 border-zinc-200 border-b bg-zinc-50/70 px-4 py-3 text-xs text-zinc-500">
        {filters.from || filters.to ? (
          <Badge tone="zinc">
            {filters.from || "start"} - {filters.to || "now"}
          </Badge>
        ) : null}
        {filters.repositoryId ? <Badge tone="zinc">{"Repository selected"}</Badge> : null}
        {filters.projectId ? <Badge tone="zinc">{"Project selected"}</Badge> : null}
        {filters.model ? <Badge tone="zinc">{filters.model}</Badge> : null}
        {filters.deviceId ? <Badge tone="zinc">{"Device selected"}</Badge> : null}
        {!activeFilters ? <span>{"No filters"}</span> : null}
      </div>
      <div
        aria-hidden={!expanded}
        className={`grid gap-3 bg-zinc-50/70 p-4 md:grid-cols-3 xl:grid-cols-7 ${
          expanded ? "" : "hidden"
        }`}
        hidden={!expanded}
      >
        <DatePicker
          max={filters.to || undefined}
          onChange={(value) => update("from", value)}
          placeholder={"From date"}
          value={filters.from}
        />
        <DatePicker
          min={filters.from || undefined}
          onChange={(value) => update("to", value)}
          placeholder={"To date"}
          value={filters.to}
        />
        <FacetSelect
          label={"Repository"}
          onChange={(value) => update("repositoryId", value)}
          options={facets?.repositories ?? []}
          value={filters.repositoryId}
        />
        <FacetSelect
          label={"Project"}
          onChange={(value) => update("projectId", value)}
          options={facets?.projects ?? []}
          value={filters.projectId}
        />
        <FacetSelect
          label={"Model"}
          onChange={(value) => update("model", value)}
          options={facets?.models ?? []}
          value={filters.model}
        />
        <FacetSelect
          label={"Device"}
          onChange={(value) => update("deviceId", value)}
          options={facets?.devices ?? []}
          value={filters.deviceId}
        />
        {showBucket ? (
          <Select
            onValueChange={(value) => onBucketChange?.(value)}
            options={[
              { label: "Daily", value: "day" },
              { label: "Weekly", value: "week" },
              { label: "Monthly", value: "month" },
            ]}
            placeholder={"Bucket"}
            value={bucket ?? "day"}
          />
        ) : null}
        <Button
          className="h-10"
          onClick={() => onFiltersChange(emptyDashboardFilters)}
          type="button"
          variant="outline"
        >
          {"Reset"}
        </Button>
      </div>
    </Panel>
  );
}

export function FacetSelect({
  label,
  onChange,
  options,
  value,
}: {
  label: string;
  onChange: (value: string) => void;
  options: FacetOption[];
  value: string;
}) {
  const labelCounts = options.reduce<Record<string, number>>((acc, option) => {
    acc[option.label] = (acc[option.label] ?? 0) + 1;
    return acc;
  }, {});

  const selectOptions: SelectOption[] = [
    { label: `${label}: all`, value: "" },
    ...options.map((option) => ({
      label: `${formatFacetLabel(option, labelCounts[option.label] > 1)}${
        option.count ? ` (${option.count})` : ""
      }`,
      value: option.id,
    })),
  ];

  return (
    <Select
      onValueChange={onChange}
      options={selectOptions}
      placeholder={`${label}: all`}
      value={value}
    />
  );
}
