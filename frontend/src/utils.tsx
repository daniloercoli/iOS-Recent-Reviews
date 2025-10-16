export function formatDate(iso: string) {
  const d = new Date(iso);
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(d);
}

export function relativeTime(iso: string) {
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const d = new Date(iso).getTime();
  const now = Date.now();
  const diffSec = Math.round((d - now) / 1000);
  const abs = Math.abs(diffSec);

  if (abs < 60) return rtf.format(Math.round(diffSec), "second");
  const diffMin = Math.round(diffSec / 60);
  if (Math.abs(diffMin) < 60) return rtf.format(diffMin, "minute");
  const diffH = Math.round(diffMin / 60);
  if (Math.abs(diffH) < 24) return rtf.format(diffH, "hour");
  const diffD = Math.round(diffH / 24);
  return rtf.format(diffD, "day");
}

export function Stars({ n }: { n: number }) {
  const clamped = Math.max(0, Math.min(5, Math.trunc(Number(n))));
  return (
    <span aria-label={`${clamped} stelle`}>
      {"★★★★★☆☆☆☆☆".slice(5 - clamped, 10 - clamped)}
    </span>
  );
}


