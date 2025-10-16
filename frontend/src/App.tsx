import { useEffect, useMemo, useState } from "react";
import { fetchApps, fetchReviews } from "./api";
import type { AppConfig, Review } from "./types";
import ReviewCard from "./components/ReviewCard";

const PRESETS = [12, 24, 48, 72, 168, 336]; // ore: 0.5, 1, 2, 3 days, 1, 2 weeks

export default function App() {
  const [apps, setApps] = useState<AppConfig[]>([]);
  const [selected, setSelected] = useState<AppConfig | null>(null);

  const [hours, setHours] = useState<number>(48);
  const [customHours, setCustomHours] = useState<string>("");

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reviews, setReviews] = useState<Review[]>([]);
  const [lastWindow, setLastWindow] = useState<{ from: string; to: string; count: number } | null>(null);

  const effectiveHours = useMemo(() => {
    const v = Number(customHours);
    if (!Number.isNaN(v) && v > 0 && v <= 24 * 90) return v;
    return hours;
  }, [customHours, hours]);

  useEffect(() => {
    fetchApps().then(setApps).catch(e => {
      console.error(e);
      setError("Impossibile caricare la lista app");
    });
  }, []);

  useEffect(() => {
    if (!selected) return;
    (async () => {
      setLoading(true); setError(null);
      try {
        const resp = await fetchReviews(selected.appId, selected.country, effectiveHours);
        setReviews(resp.reviews);
        setLastWindow({ from: resp.from, to: resp.to, count: resp.count });
      } catch (e: any) {
        console.error(e);
        setError("Errore nel recupero recensioni");
      } finally {
        setLoading(false);
      }
    })();
  }, [selected, effectiveHours]);

  // Update the tile after defining selected (chosen app) and hours/effectiveHours:
  useEffect(() => {
    const base = "iOS Recent App Store Reviews";
    if (!selected) {
      document.title = base;
      return;
    }
    // ie: "Reviews · 595068606 (us) · last 48h"
    document.title = `Reviews · ${selected.appId} (${selected.country}) · last ${effectiveHours}h`;
  }, [selected, effectiveHours]);

  return (
    <div style={{ padding: 24, fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, sans-serif" }}>
      <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 12 }}>
        iOS Recent App Store Reviews
      </h1>

      {/* Selectors */}
      <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center", marginBottom: 12 }}>
        <label>
          App:
          <select
            style={{ marginLeft: 8, padding: 6 }}
            value={selected ? `${selected.appId}:${selected.country}` : ""}
            onChange={(e) => {
              const [appId, country] = e.target.value.split(":");
              const app = apps.find(a => a.appId === appId && a.country === country) || null;
              setSelected(app);
            }}
          >
            <option value="">— Select one app —</option>
            {apps.map(a => (
              <option key={`${a.appId}:${a.country}`} value={`${a.appId}:${a.country}`}>
                {a.appId} ({a.country})
              </option>
            ))}
          </select>
        </label>

        <label>
          Interval:
          <select
            style={{ marginLeft: 8, padding: 6 }}
            value={hours}
            onChange={(e) => setHours(Number(e.target.value))}
          >
            {PRESETS.map(h => (
              <option key={h} value={h}>{h}h</option>
            ))}
          </select>
        </label>

        <label title="Custom hrs (till to 2160)">
          Custom (h):
          <input
            type="number" min={1} max={2160}
            placeholder="es. 6"
            value={customHours}
            onChange={(e) => setCustomHours(e.target.value)}
            style={{ width: 90, marginLeft: 8, padding: 6 }}
          />
        </label>

        <button
          onClick={() => {
            // retrigger fetch
            setSelected(s => s ? ({ ...s }) : s);
          }}
          disabled={!selected || loading}
          style={{ padding: "8px 12px", borderRadius: 8, border: "1px solid #ccc", background: "#f8f8f8" }}
        >
          {loading ? "Laoding..." : "Refresh"}
        </button>

        {lastWindow && (
          <span style={{ marginLeft: 8, fontSize: 12, color: "#666" }}>
            Window: {new Date(lastWindow.from).toLocaleString()} → {new Date(lastWindow.to).toLocaleString()} — {lastWindow.count} rec.
          </span>
        )}
      </div>

      {/* State */}
      {!selected && <p>Select one app</p>}
      {error && <p style={{ color: "crimson" }}>{error}</p>}

      {/* Lista recensioni */}
      <div>
        {reviews.length === 0 && selected && !loading && !error && (
          <p>No reviews in the selected timeframe.</p>
        )}
        {reviews.map(r => <ReviewCard key={r.id} r={r} />)}
      </div>
    </div>
  );
}
