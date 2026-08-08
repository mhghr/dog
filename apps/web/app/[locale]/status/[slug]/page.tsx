import type { Metadata } from "next";
import { SearchX } from "lucide-react";
import { getTranslations, setRequestLocale } from "next-intl/server";

import { AutoRefresh } from "@/shared/ui/auto-refresh";
import { RelativeTime } from "@/shared/ui/relative-time";
import { BrandMark } from "@/shared/ui/brand-mark";
import { formatPercent } from "@/shared/utils/formatters";
import { cn } from "@/shared/utils/cn";
import type { PublicStatusPage } from "@/features/status-pages/model/types";

const OVERALL_STYLES: Record<
  PublicStatusPage["status"],
  { container: string; dot: string }
> = {
  operational: {
    container: "border-success/40 bg-success/10 text-success",
    dot: "bg-success",
  },
  partial_outage: {
    container: "border-warning/40 bg-warning/10 text-warning",
    dot: "bg-warning",
  },
  major_outage: {
    container: "border-destructive/40 bg-destructive/10 text-destructive",
    dot: "bg-destructive",
  },
};

const COMPONENT_DOT: Record<string, string> = {
  up: "bg-success",
  down: "bg-destructive",
  paused: "bg-warning",
  unknown: "bg-muted-foreground/60",
};

async function fetchStatusPage(slug: string): Promise<PublicStatusPage | null> {
  const baseURL =
    process.env.API_INTERNAL_URL ??
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    "http://localhost:5000";

  try {
    const response = await fetch(
      `${baseURL}/api/v1/status-pages/public/${encodeURIComponent(slug)}`,
      { next: { revalidate: 30 } },
    );
    if (!response.ok) {
      return null;
    }
    return (await response.json()) as PublicStatusPage;
  } catch {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; slug: string }>;
}): Promise<Metadata> {
  const { locale, slug } = await params;
  const t = await getTranslations({ locale, namespace: "statusPage" });
  const page = await fetchStatusPage(slug);

  return { title: page ? `${page.name} — ${t("title")}` : t("title") };
}

function formatUptime(value: number | null, locale: string) {
  return formatPercent(value, locale);
}

export default async function PublicStatusPageRoute({
  params,
}: {
  params: Promise<{ locale: string; slug: string }>;
}) {
  const { locale, slug } = await params;
  setRequestLocale(locale);

  const t = await getTranslations({ locale, namespace: "statusPage" });
  const page = await fetchStatusPage(slug);

  if (!page) {
    return (
      <main className="flex min-h-screen items-center justify-center px-4">
        <div className="flex max-w-sm flex-col items-center gap-3 text-center">
          <SearchX className="size-8 text-muted-foreground" aria-hidden />
          <h1 className="text-lg font-semibold">{t("notFoundTitle")}</h1>
          <p className="text-sm text-muted-foreground">{t("notFoundBody")}</p>
        </div>
      </main>
    );
  }

  const overall = OVERALL_STYLES[page.status];

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-2xl flex-col px-4 py-12">
      <AutoRefresh />

      <header className="flex items-center gap-3">
        <BrandMark />
        <div className="min-w-0">
          <h1 className="truncate text-lg font-semibold tracking-tight">
            {page.name}
          </h1>
          {page.description ? (
            <p className="truncate text-sm text-muted-foreground">
              {page.description}
            </p>
          ) : null}
        </div>
      </header>

      <div
        className={cn(
          "mt-8 flex items-center gap-3 rounded-xl border px-4 py-3.5 font-medium",
          overall.container,
        )}
      >
        <span className={cn("size-2.5 rounded-full", overall.dot)} aria-hidden />
        {t(page.status)}
      </div>

      <section className="mt-6 overflow-hidden rounded-xl border border-border bg-card">
        <div className="flex items-center justify-between border-b border-border/70 px-4 py-2.5 text-xs text-muted-foreground">
          <span>{t("uptimeLabel")}</span>
          <div className="flex items-center gap-4" dir="ltr">
            <span className="w-16 text-end">{t("range24h")}</span>
            <span className="hidden w-16 text-end sm:inline">{t("range7d")}</span>
            <span className="hidden w-16 text-end sm:inline">{t("range30d")}</span>
          </div>
        </div>
        <ul className="divide-y divide-border/70">
          {page.components.map((component, index) => (
            <li
              key={`${component.name}-${index}`}
              className="flex items-center justify-between gap-3 px-4 py-3"
            >
              <div className="flex min-w-0 items-center gap-2.5">
                <span
                  className={cn(
                    "size-2 shrink-0 rounded-full",
                    COMPONENT_DOT[component.status] ?? COMPONENT_DOT.unknown,
                  )}
                  aria-hidden
                />
                <span className="truncate text-sm font-medium">
                  {component.name}
                </span>
              </div>
              <div
                className="flex shrink-0 items-center gap-4 text-sm tabular-nums text-muted-foreground"
                dir="ltr"
              >
                <span className="w-16 text-end">
                  {formatUptime(component.uptime_24h, locale)}
                </span>
                <span className="hidden w-16 text-end sm:inline">
                  {formatUptime(component.uptime_7d, locale)}
                </span>
                <span className="hidden w-16 text-end sm:inline">
                  {formatUptime(component.uptime_30d, locale)}
                </span>
              </div>
            </li>
          ))}
        </ul>
      </section>

      <footer className="mt-6 text-xs text-muted-foreground">
        {t("lastUpdated")}: <RelativeTime value={page.checked_at} />
      </footer>
    </main>
  );
}
