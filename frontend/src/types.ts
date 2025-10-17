export type Review = {
    id: string;
    appId: string;
    country: string;
    author: string;
    rating: number;      // 1–5
    title: string;
    content: string;
    submittedAt: string; // ISO UTC
};

export type AppConfig = { appId: string; country: string, name?: string };
