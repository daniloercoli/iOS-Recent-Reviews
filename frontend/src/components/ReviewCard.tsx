import type { Review } from "../types";

function formatRelative(iso: string) {
    const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
    const t = new Date(iso).getTime();
    const now = Date.now();
    const diffSec = Math.round((t - now) / 1000);
    const abs = Math.abs(diffSec);

    if (abs < 60) return rtf.format(diffSec, "second");
    const diffMin = Math.round(diffSec / 60);
    if (Math.abs(diffMin) < 60) return rtf.format(diffMin, "minute");
    const diffH = Math.round(diffMin / 60);
    if (Math.abs(diffH) < 24) return rtf.format(diffH, "hour");
    const diffD = Math.round(diffH / 24);
    return rtf.format(diffD, "day");
}

export default function ReviewCard({ r }: { r: Review }) {
    const relative = formatRelative(r.submittedAt);
    const hasTitle = !!(r.title && r.title.trim().length);

    // iniziale autore per l’avatar
    const initial = (r.author || "A").trim().charAt(0).toUpperCase();

    return (
        <article className="review-card" role="article" aria-label="App Store Review">
            <header className="review-card__header">
                <div className="review-card__avatar" aria-hidden="true">{initial}</div>
                <div className="review-card__meta">
                    <div className="review-card__author" title={r.author || "Anonimo"}>
                        {r.author || "Anonimo"}
                    </div>
                    <div className="review-card__time" title={r.submittedAt}>
                        {relative} <span className="review-card__time-iso">· {r.submittedAt}</span>
                    </div>
                </div>
                <div className="review-card__rating" aria-label={`${r.rating} su 5`}>
                    <Stars n={r.rating} />
                    <span className="review-card__rating-badge">{r.rating}/5</span>
                </div>
            </header>

            {hasTitle && (
                <h3 className="review-card__title" title={r.title}>
                    {r.title}
                </h3>
            )}
            <div className="review-card__content">
                {r.content?.trim() || "—"}
            </div>
        </article>
    );
}


function Stars(props: { n: number }) {
    const n = Math.max(0, Math.min(5, Number(props.n) || 0));
    return <span className="stars">{'★'.repeat(n) + '☆'.repeat(5 - n)}</span>;
}