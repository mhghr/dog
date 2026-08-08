import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";

import { Button } from "@/shared/ui/button";
import { MockCard, MockListRow, SectionHeader, StatusChip } from "@/features/marketing/marketing";
import { Link } from "@/i18n/navigation";
import { MONITOR_TYPE_ICONS, MONITOR_TYPES } from "@/entities/monitor/model/monitor-meta";
import { cn } from "@/shared/utils/cn";

function FeatureRow({
  eyebrow,
  title,
  body,
  children,
  reverse,
}: {
  eyebrow: string;
  title: string;
  body: string;
  children: React.ReactNode;
  reverse?: boolean;
}) {
  return (
    <div className="grid items-center gap-12 lg:grid-cols-2 lg:gap-16">
      <div className={cn(reverse && "lg:order-2")}>
        <p className="text-sm font-semibold text-primary">{eyebrow}</p>
        <h3 className="mt-2 text-pretty text-xl font-bold tracking-tight sm:text-2xl">
          {title}
        </h3>
        <p className="mt-3 text-pretty leading-relaxed text-muted-foreground">
          {body}
        </p>
      </div>
      <div className={cn(reverse && "lg:order-1")}>{children}</div>
    </div>
  );
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketing" });

  return {
    title: t("features.title"),
    description: t("features.subtitle"),
    alternates: {
      canonical: `/${locale}/features`,
      languages: { en: "/en/features", fa: "/fa/features" },
    },
  };
}

export default async function FeaturesPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);

  const t = await getTranslations({ locale, namespace: "marketing" });
  const tTypes = await getTranslations({ locale, namespace: "types" });
  const tLanding = await getTranslations({ locale, namespace: "landing" });

  return (
    <div>
      <section className="relative overflow-hidden">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-grid-faint [mask-image:radial-gradient(ellipse_75%_65%_at_50%_0%,black,transparent)]"
        />
        <div className="relative mx-auto w-full max-w-7xl px-4 pb-20 pt-16 lg:pb-28 lg:pt-24">
          <SectionHeader
            eyebrow={t("features.heroEyebrow")}
            title={t("features.heroTitle")}
            subtitle={t("features.heroSubtitle")}
          />
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <SectionHeader
          eyebrow={t("features.typesEyebrow")}
          title={t("features.typesTitle")}
          subtitle={t("features.typesSubtitle")}
        />

        <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-4">
          {MONITOR_TYPES.map((monitorType) => {
            const Icon = MONITOR_TYPE_ICONS[monitorType];
            return (
              <div
                key={monitorType}
                className="group flex flex-col gap-4 bg-card p-5 transition-colors hover:bg-accent/40"
              >
                <span className="grid size-9 place-items-center rounded-lg border border-border/70 bg-background text-primary transition-colors group-hover:border-primary/40">
                  <Icon className="size-4.5" aria-hidden />
                </span>
                <div>
                  <h3 className="text-sm font-semibold">
                    {tTypes(monitorType)}
                  </h3>
                  <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                    {tLanding(`typeDesc.${monitorType}`)}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <FeatureRow
          eyebrow={t("features.liveEyebrow")}
          title={t("features.liveTitle")}
          body={t("features.liveBody")}
        >
          <div
            aria-hidden
            dir="ltr"
            className="overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm"
          >
            <div className="flex items-center justify-between border-b border-border/70 pb-3">
              <span className="font-mono text-caption font-medium tracking-wider text-muted-foreground">
                SSE STREAM
              </span>
              <span className="flex items-center gap-1.5 font-mono text-caption font-medium tracking-wider text-success">
                <span className="relative flex size-1.5">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-60" />
                  <span className="relative inline-flex size-1.5 rounded-full bg-success" />
                </span>
                CONNECTED
              </span>
            </div>
            <div className="pt-3 font-mono text-caption-lg leading-6">
              {[
                { ok: true, ts: "14:32:01", type: "http", target: "api.example.com", result: "200 · 142ms" },
                { ok: true, ts: "14:32:02", type: "ping", target: "203.0.113.10", result: "23ms" },
                { ok: false, ts: "14:32:04", type: "smtp", target: "mail.example.com", result: "starttls failed" },
                { ok: true, ts: "14:32:06", type: "tls", target: "example.com", result: "valid · 61d" },
                { ok: true, ts: "14:32:08", type: "dns", target: "example.com", result: "A 93.184.216.34" },
              ].map((line) => (
                <p key={line.ts} className="whitespace-pre text-muted-foreground">
                  <span className="inline-block w-14 text-muted-foreground/60">
                    {line.ts}
                  </span>
                  <span
                    className={cn(
                      "mr-1 inline-block w-8",
                      line.ok ? "text-success" : "text-destructive",
                    )}
                  >
                    {line.ok ? "OK" : "ERR"}
                  </span>
                  <span className="text-muted-foreground/70">{line.type}</span>{" "}
                  <span>{line.target}</span>{" "}
                  <span className={line.ok ? "text-muted-foreground/70" : "text-destructive"}>
                    {line.result}
                  </span>
                </p>
              ))}
            </div>
          </div>
        </FeatureRow>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <FeatureRow
          eyebrow={t("features.locationsEyebrow")}
          title={t("features.locationsTitle")}
          body={t("features.locationsBody")}
          reverse
        >
          <div
            aria-hidden
            dir="ltr"
            className="overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm"
          >
            <p className="font-mono text-caption font-medium tracking-wider text-muted-foreground">
              PROBE RESULTS BY LOCATION
            </p>
            <div className="mt-3 space-y-2">
              {[
                { location: "FRA", latency: "142 ms", status: "up" },
                { location: "AMS", latency: "156 ms", status: "up" },
                { location: "THR", latency: "231 ms", status: "down" },
              ].map((row) => (
                <div
                  key={row.location}
                  className="flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2.5"
                >
                  <div className="flex items-center gap-2.5">
                    <span
                      className={cn(
                        "size-2 rounded-full",
                        row.status === "up" ? "bg-success" : "bg-destructive",
                      )}
                    />
                    <span className="font-mono text-xs font-medium">
                      {row.location}
                    </span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="font-mono text-xs tabular-nums text-muted-foreground">
                      {row.latency}
                    </span>
                    <span
                      className={cn(
                        "rounded-full px-2 py-0.5 font-mono text-caption-sm font-medium",
                        row.status === "up"
                          ? "bg-success/10 text-success"
                          : "bg-destructive/10 text-destructive",
                      )}
                    >
                      {row.status === "up" ? "UP" : "DOWN"}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </FeatureRow>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <FeatureRow
          eyebrow={t("features.chartsEyebrow")}
          title={t("features.chartsTitle")}
          body={t("features.chartsBody")}
        >
          <div
            aria-hidden
            dir="ltr"
            className="overflow-hidden rounded-xl border border-border bg-card shadow-sm"
          >
            <div className="flex items-center gap-0.5 border-b border-border/70 px-4 py-2.5">
              {["24h", "7d", "30d"].map((range, i) => (
                <span
                  key={range}
                  className={cn(
                    "rounded-md px-2.5 py-0.5 font-mono text-caption font-medium",
                    i === 0
                      ? "bg-muted text-foreground"
                      : "text-muted-foreground",
                  )}
                >
                  {range}
                </span>
              ))}
            </div>
            <div className="px-4 py-4">
              <p className="font-mono text-caption font-medium tracking-wider text-muted-foreground">
                P95 LATENCY
              </p>
              <p className="mt-1 font-mono text-xl font-semibold tabular-nums">
                231 ms
              </p>
              <svg
                viewBox="0 0 300 72"
                className="mt-3 h-16 w-full text-primary"
                preserveAspectRatio="none"
              >
                <line
                  x1="0" y1="48" x2="300" y2="48"
                  className="stroke-border/60"
                  strokeWidth="1"
                />
                <line
                  x1="0" y1="24" x2="300" y2="24"
                  className="stroke-border/40"
                  strokeWidth="1"
                  strokeDasharray="3 3"
                />
                <path
                  d="M0 40 C20 38 28 44 44 42 C60 40 68 48 84 44 C100 40 108 50 124 46 C140 42 148 52 164 48 C180 44 188 56 204 50 C220 46 228 54 244 52 C260 50 268 60 284 54 C300 50 308 58 320 52"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinejoin="round"
                />
                <circle cx="164" cy="48" r="3" fill="currentColor" />
              </svg>
              <div className="mt-2 flex items-center gap-4">
                <div className="flex items-center gap-1.5">
                  <span className="size-2 rounded-full bg-primary" />
                  <span className="font-mono text-caption-sm text-muted-foreground">
                    P95
                  </span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="size-2 rounded-full bg-success" />
                  <span className="font-mono text-caption-sm text-muted-foreground">
                    AVG 187 ms
                  </span>
                </div>
              </div>
            </div>
          </div>
        </FeatureRow>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <FeatureRow
          eyebrow={t("features.schedulingEyebrow")}
          title={t("features.schedulingTitle")}
          body={t("features.schedulingBody")}
          reverse
        >
          <div
            aria-hidden
            dir="ltr"
            className="overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm"
          >
            <p className="font-mono text-caption font-medium tracking-wider text-muted-foreground">
              MONITOR CONFIG
            </p>
            <div className="mt-3 space-y-1.5">
              {[
                { label: "Interval", value: "60s" },
                { label: "Timeout", value: "5 000 ms" },
                { label: "Retries", value: "3" },
                { label: "Paused", value: "false" },
              ].map((row) => (
                <div
                  key={row.label}
                  className="flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2.5"
                >
                  <span className="font-mono text-caption tracking-wider text-muted-foreground">
                    {row.label}
                  </span>
                  <span className="font-mono text-xs font-medium tabular-nums">
                    {row.value}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </FeatureRow>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <FeatureRow
          eyebrow={t("features.ssrfEyebrow")}
          title={t("features.ssrfTitle")}
          body={t("features.ssrfBody")}
        >
          <div
            aria-hidden
            dir="ltr"
            className="overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm"
          >
            <p className="font-mono text-caption font-medium tracking-wider text-muted-foreground">
              BLOCKED TARGETS
            </p>
            <div className="mt-3 space-y-1">
              {[
                "10.0.0.0/8",
                "172.16.0.0/12",
                "192.168.0.0/16",
                "169.254.0.0/16",
                "127.0.0.0/8",
                "fd00::/8",
              ].map((range) => (
                <div
                  key={range}
                  className="flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2"
                >
                  <span className="font-mono text-caption-lg">
                    {range}
                  </span>
                  <span className="font-mono text-caption font-medium text-destructive">
                    BLOCKED
                  </span>
                </div>
              ))}
            </div>
          </div>
        </FeatureRow>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <FeatureRow
          eyebrow={t("features.statusPagesEyebrow")}
          title={t("features.statusPagesTitle")}
          body={t("features.statusPagesBody")}
          reverse
        >
          <div
            aria-hidden
            dir="ltr"
            className="overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm"
          >
            <p className="font-mono text-caption font-medium tracking-wider text-muted-foreground">
              /STATUS/MAIN-STATUS
            </p>
            <div className="mt-3 space-y-2">
              {[
                { name: "API Gateway", status: "up" },
                { name: "Database cluster", status: "up" },
                { name: "Mail delivery", status: "down" },
              ].map((comp) => (
                <div
                  key={comp.name}
                  className="flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2.5"
                >
                  <div className="flex items-center gap-2.5">
                    <span
                      className={cn(
                        "size-2 rounded-full",
                        comp.status === "up" ? "bg-success" : "bg-destructive",
                      )}
                    />
                    <span className="font-mono text-xs">{comp.name}</span>
                  </div>
                  <span
                    className={cn(
                      "rounded-full px-2 py-0.5 font-mono text-caption-sm font-medium",
                      comp.status === "up"
                        ? "bg-success/10 text-success"
                        : "bg-destructive/10 text-destructive",
                    )}
                  >
                    {comp.status === "up" ? "OPERATIONAL" : "DOWN"}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </FeatureRow>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <FeatureRow
          eyebrow={t("features.i18nEyebrow")}
          title={t("features.i18nTitle")}
          body={t("features.i18nBody")}
        >
          <div
            aria-hidden
            dir="ltr"
            className="grid grid-cols-2 gap-4"
          >
            <div className="rounded-xl border border-border bg-card p-4">
              <p className="font-mono text-caption-sm font-medium tracking-wider text-muted-foreground">
                LIGHT · EN
              </p>
              <div className="mt-3 space-y-2">
                <div className="h-2 w-3/4 rounded bg-muted" />
                <div className="h-2 w-full rounded bg-muted" />
                <div className="h-2 w-1/2 rounded bg-muted" />
              </div>
            </div>
            <div className="rounded-xl border border-border bg-card p-4 dark">
              <p className="font-mono text-caption-sm font-medium tracking-wider text-muted-foreground">
                DARK · FA
              </p>
              <div className="mt-3 space-y-2" dir="rtl">
                <div className="h-2 w-1/2 rounded bg-muted" />
                <div className="h-2 w-full rounded bg-muted" />
                <div className="h-2 w-3/4 rounded bg-muted" />
              </div>
            </div>
          </div>
        </FeatureRow>
      </section>

      <section className="relative overflow-hidden border-t border-border/70">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-grid-faint [mask-image:radial-gradient(ellipse_55%_70%_at_50%_100%,black,transparent)]"
        />
        <div className="relative mx-auto flex w-full max-w-7xl flex-col items-center gap-6 px-4 py-20 text-center lg:py-28">
          <h2 className="max-w-2xl text-balance text-3xl font-bold tracking-tight lg:text-4xl">
            {t("features.cta")}
          </h2>
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Button asChild size="lg" className="gap-2 px-5">
              <Link href="/login">
                {t("features.cta")}
              </Link>
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">
            {t("features.ctaNote")}
          </p>
        </div>
      </section>
    </div>
  );
}
