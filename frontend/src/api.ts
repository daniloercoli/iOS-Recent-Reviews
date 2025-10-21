const API_BASE = import.meta.env.VITE_API_BASE ?? "http://localhost:8080";

export async function fetchApps(): Promise<{ appId: string; country: string }[]> {
    const res = await fetch(`${API_BASE}/apps`);
    if (!res.ok) throw new Error(`/apps ${res.status}`);
    return res.json();
}

export async function fetchReviews(appId: string, country: string, hours: number, minRating?: number) {
    const url = new URL(`${API_BASE}/reviews`);
    url.searchParams.set("appId", appId);
    url.searchParams.set("country", country);
    url.searchParams.set("hours", String(hours));
    if (minRating && minRating >= 1 && minRating <= 5) {
        url.searchParams.set("minRating", String(minRating));
    }
    const res = await fetch(url.toString());
    if (!res.ok) throw new Error(`/reviews ${res.status}`);
    return res.json() as Promise<{
        appId: string;
        country: string;
        from: string;
        to: string;
        count: number;
        reviews: import("./types").Review[];
    }>;
}
