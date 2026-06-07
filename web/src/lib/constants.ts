import type { LucideIcon } from "lucide-react";
import { Activity, Archive, BarChart3, Download, MessageSquareText, Search } from "lucide-react";
import type { DashboardFilters } from "@/types";

export const navItems: { to: string; label: string; icon: LucideIcon; end?: boolean }[] = [
  { to: "/", label: "Overview", icon: Activity, end: true },
  { to: "/usage", label: "Usage", icon: BarChart3 },
  { to: "/sessions", label: "Sessions", icon: MessageSquareText },
  { to: "/search", label: "Search", icon: Search },
  { to: "/exports", label: "Exports", icon: Download },
  { to: "/archive", label: "Archive", icon: Archive },
];

export const emptyDashboardFilters: DashboardFilters = {
  deviceId: "",
  from: "",
  model: "",
  projectId: "",
  repositoryId: "",
  to: "",
};

export const fieldClassName =
  "h-10 min-w-0 rounded-none border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-zinc-400 focus:ring-4 focus:ring-zinc-100";
