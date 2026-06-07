import { Activity, LogOut, PanelLeft, RefreshCw, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { NavLink, Outlet } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { navItems } from "@/lib/constants";

export function DashboardLayout({ onLogout }: { onLogout: () => void }) {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <main className="min-h-screen bg-background text-zinc-950">
      <aside
        className="fixed inset-y-0 left-0 hidden w-[280px] p-3 transition-transform duration-200 ease-out lg:block"
        style={{ transform: sidebarOpen ? "translateX(0)" : "translateX(-100%)" }}
      >
        <div className="flex h-full flex-col rounded-none border border-zinc-200 bg-white">
          <div className="flex h-[68px] items-center border-zinc-200 border-b p-3">
            <NavLink
              aria-label="Go to overview"
              className="flex min-w-0 items-center gap-3 rounded-none px-1 py-1 text-zinc-950 transition-colors hover:text-zinc-600"
              title="Go to overview"
              to="/"
            >
              <div className="flex size-10 items-center justify-center rounded-none bg-zinc-950 text-white">
                <Activity size={18} />
              </div>
              <div className="min-w-0">
                <div className="truncate font-semibold text-sm">Codex Usage</div>
              </div>
            </NavLink>
          </div>

          <nav className="flex-1 space-y-1 p-3 text-sm">
            <div className="px-2 pb-2 font-medium text-[11px] text-zinc-400 uppercase tracking-wide">
              {"Workspace"}
            </div>
            {navItems.map(({ to, label, icon: Icon, end }) => (
              <NavLink
                className={({ isActive }) =>
                  `flex h-9 items-center gap-2 rounded-none px-2.5 font-medium transition-colors ${
                    isActive
                      ? "bg-zinc-950 text-white"
                      : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950"
                  }`
                }
                end={end}
                key={to}
                to={to}
              >
                <Icon size={16} />
                {label}
              </NavLink>
            ))}
          </nav>

          <div className="border-zinc-200 border-t p-3">
            <div className="rounded-none bg-zinc-50 p-3">
              <div className="flex items-center gap-2 text-zinc-700 text-xs">
                <ShieldCheck size={14} />
                {"Local session auth"}
              </div>
              <Button
                className="mt-3 w-full"
                onClick={onLogout}
                size="default"
                type="button"
                variant="outline"
              >
                <LogOut size={14} />
                {"Logout"}
              </Button>
            </div>
          </div>
        </div>
      </aside>

      <section
        className={`transition-[padding] duration-200 ease-out ${
          sidebarOpen ? "lg:pl-[280px]" : "lg:pl-0"
        }`}
      >
        <header className="sticky top-0 z-10 min-h-20 border-zinc-200 border-b bg-white/85 px-3 py-2 backdrop-blur md:px-6 md:py-3">
          <div className="mx-auto flex max-w-[1440px] flex-wrap items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <Button
                aria-label={sidebarOpen ? "Hide sidebar" : "Show sidebar"}
                aria-pressed={sidebarOpen}
                className="hidden text-muted-foreground lg:inline-flex"
                onClick={() => setSidebarOpen((open) => !open)}
                size="icon"
                title={sidebarOpen ? "Hide sidebar" : "Show sidebar"}
                type="button"
                variant="outline"
              >
                <PanelLeft size={15} />
              </Button>
              <NavLink
                aria-label="Go to overview"
                className="min-w-0 text-zinc-950 transition-colors hover:text-zinc-600"
                title="Go to overview"
                to="/"
              >
                <h1 className="truncate font-semibold text-lg">Codex Usage</h1>
                <p className="hidden truncate text-sm text-zinc-500 sm:block">
                  {"Unified view of tokens, sessions, search, and raw archives"}
                </p>
              </NavLink>
            </div>
            <div className="flex items-center gap-2">
              <Button
                aria-label={"Open navigation menu"}
                aria-expanded={mobileMenuOpen}
                className="lg:hidden"
                onClick={() => setMobileMenuOpen((open) => !open)}
                size="icon-lg"
                type="button"
                variant="outline"
                title={"Open navigation menu"}
              >
                <PanelLeft size={16} />
              </Button>
              <Button
                onClick={() => window.location.reload()}
                size="default"
                type="button"
                variant="outline"
              >
                <RefreshCw size={16} />
                {"Refresh"}
              </Button>
              <Button
                aria-label={"Logout"}
                className="lg:hidden"
                onClick={onLogout}
                size="icon-lg"
                type="button"
                variant="outline"
                title={"Logout"}
              >
                <LogOut size={16} />
              </Button>
            </div>
          </div>

          <nav
            className={`mx-auto mt-2 max-w-[1440px] border border-zinc-200 bg-white p-1 text-sm lg:hidden ${
              mobileMenuOpen ? "grid" : "hidden"
            }`}
          >
            {navItems.map(({ to, label, icon: Icon, end }) => (
              <NavLink
                className={({ isActive }) =>
                  `flex h-9 items-center gap-2 rounded-none px-3 font-medium ${
                    isActive ? "bg-zinc-950 text-white" : "text-zinc-700 hover:bg-zinc-100"
                  }`
                }
                end={end}
                key={to}
                onClick={() => setMobileMenuOpen(false)}
                to={to}
              >
                <Icon size={15} />
                {label}
              </NavLink>
            ))}
          </nav>
        </header>

        <div className="mx-auto min-w-0 max-w-[1440px] p-3 md:p-6">
          <Outlet />
        </div>
      </section>
    </main>
  );
}
