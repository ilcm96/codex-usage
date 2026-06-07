import { Activity } from "lucide-react";
import type { FormEvent } from "react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { apiBase } from "@/lib/api";

export function LoginPage({ onAuthenticated }: { onAuthenticated: () => void }) {
  const [password, setPassword] = useState("");
  const [loginError, setLoginError] = useState("");

  async function login(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoginError("");
    const res = await fetch(`${apiBase}/api/auth/login`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password }),
    });
    if (!res.ok) {
      setLoginError("The password is incorrect.");
      return;
    }
    onAuthenticated();
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background p-4 text-zinc-950">
      <form
        className="w-full max-w-sm rounded-none border border-zinc-200 bg-white/95 p-6 backdrop-blur"
        onSubmit={login}
      >
        <div className="flex items-center gap-2">
          <div className="flex size-10 items-center justify-center rounded-none bg-zinc-950 text-white">
            <Activity size={18} />
          </div>
          <div>
            <h1 className="font-semibold text-base">Codex Usage</h1>
            <p className="text-sm text-zinc-500">{"Admin session"}</p>
          </div>
        </div>
        <input
          className="mt-6 h-10 w-full rounded-none border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-zinc-400 focus:ring-4 focus:ring-zinc-100"
          onChange={(event) => setPassword(event.target.value)}
          placeholder="Admin password"
          type="password"
          value={password}
        />
        {loginError ? <p className="mt-2 text-sm text-zinc-600">{loginError}</p> : null}
        <Button className="mt-4 w-full" size="lg" type="submit">
          {"Login"}
        </Button>
      </form>
    </main>
  );
}
