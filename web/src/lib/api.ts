import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

export const apiBase = import.meta.env.VITE_API_BASE_URL ?? "http://127.0.0.1:8080";

export function useApiData<T>(path: string) {
  const navigate = useNavigate();
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(Boolean(path));
  const [error, setError] = useState("");

  useEffect(() => {
    if (!path) {
      setData(null);
      setLoading(false);
      setError("");
      return;
    }

    let cancelled = false;

    async function run() {
      setLoading(true);
      setError("");
      try {
        const res = await fetch(`${apiBase}${path}`, { credentials: "include" });
        if (res.status === 401) {
          navigate("/", { replace: true });
          window.location.reload();
          return;
        }
        if (!res.ok) {
          throw new Error(`API request failed: ${res.status}`);
        }
        const value = (await res.json()) as T;
        if (!cancelled) {
          setData(value);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "API request failed");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void run();

    return () => {
      cancelled = true;
    };
  }, [navigate, path]);

  return { data, error, loading };
}
