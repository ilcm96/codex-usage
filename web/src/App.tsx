import { useEffect, useState } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";

import { LoadingScreen } from "@/components/common/primitives";
import { DashboardLayout } from "@/components/layout/DashboardLayout";
import { apiBase } from "@/lib/api";
import { ArchivePage } from "@/pages/ArchivePage";
import { ExportsPage } from "@/pages/ExportsPage";
import { LoginPage } from "@/pages/LoginPage";
import { OverviewPage } from "@/pages/OverviewPage";
import { SearchPage } from "@/pages/SearchPage";
import { SessionDetailPage } from "@/pages/SessionDetailPage";
import { SessionsPage } from "@/pages/SessionsPage";
import { UsagePage } from "@/pages/UsagePage";

function App() {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function run() {
      const res = await fetch(`${apiBase}/api/auth/me`, { credentials: "include" });
      if (!cancelled) {
        setAuthenticated(res.ok);
      }
    }

    void run();

    return () => {
      cancelled = true;
    };
  }, []);

  async function logout() {
    await fetch(`${apiBase}/api/auth/logout`, { method: "POST", credentials: "include" });
    setAuthenticated(false);
  }

  if (authenticated === null) {
    return <LoadingScreen />;
  }

  if (!authenticated) {
    return <LoginPage onAuthenticated={() => setAuthenticated(true)} />;
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route element={<DashboardLayout onLogout={logout} />}>
          <Route index element={<OverviewPage />} />
          <Route path="usage" element={<UsagePage />} />
          <Route path="sessions" element={<SessionsPage />} />
          <Route path="sessions/:sessionId" element={<SessionDetailPage />} />
          <Route path="search" element={<SearchPage />} />
          <Route path="exports" element={<ExportsPage />} />
          <Route path="archive" element={<ArchivePage />} />
          <Route path="*" element={<Navigate replace to="/" />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
