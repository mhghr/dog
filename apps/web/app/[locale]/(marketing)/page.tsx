import type { LucideIcon } from "lucide-react";
import { Activity, BellRing, Clock3, Globe, MailCheck, Radio, ShieldCheck } from "lucide-react";
import { getTranslations, setRequestLocale } from "next-intl/server";

import { BentoTile, SectionHeader } from "@/components/common/marketing";
import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";
import { MONITOR_TYPE_ICONS, MONITOR_TYPES } from "@/lib/monitor-meta";
import { cn } from "@/lib/utils";

/* Hallmark · v1.1.0 · genre: modern-minimal · macrostructure: Operations Console
 * theme: Cobalt signal glass · enrichment: Tier-A CSS product mockup
 * contrast: pass (40-41) · chrome: pass (47) · tokens: pass (48) · mobile: pass (34,49-57)
 * pre-emit critique: P5 H5 E4 S5 R4 V5
 */

const SIGNAL_ROWS: {
  icon: LucideIcon;
  host: string;
  kind: string;
  latency: string;
  status: "up" | "down" | "warn";
}[] = [
  { icon: Globe, host: "api.production.local", kind: "HTTP", latency: "142 ms", status: "up" },
  { icon: ShieldCheck, host: "app.company.com", kind: "TLS", latency: "61 d", status: "warn" },
  { icon: Radio, host: "edge-router-01", kind: "PING", latency: "23 ms", status: "up" },
  { icon: MailCheck, host: "mail.company.com", kind: "SMTP", latency: "failed", status: "down" },
  { icon: Clock3, host: "time.internal.net", kind: "NTP", latency: "±4 ms", status: "up" },
];

const INCIDENTS = [
  { time: "08:12", title: "SMTP STARTTLS failed", tone: "down" },
  { time: "08:13", title: "Retry confirmed failure", tone: "down" },
  { time: "08:14", title: "Status page component marked degraded", tone: "warn" },
  { time: "08:18", title: "HTTP and DNS probes still healthy", tone: "up" },
];

const LATENCY_BARS = [36, 44, 41, 58, 52, 69, 47, 55, 82, 49, 43, 61, 39, 46, 73, 51];

const COMMANDS = [
  "http  api.production.local  200  142ms",
  "tls   app.company.com       expires in 61d",
  "dns   company.com           A resolved",
  "smtp  mail.company.com      starttls failed",
];

function StatusDot({ status }: { status: "up" | "down" | "warn" }) {
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full",
        status === "up" && "bg-success shadow-[0_0_0_4px_color-mix(in_oklch,var(--success)_16%,transparent)]",
        status === "down" && "bg-destructive shadow-[0_0_0_4px_color-mix(in_oklch,var(--destructive)_14%,transparent)]",
        status === "warn" && "bg-warning shadow-[0_0_0_4px_color-mix(in_oklch,var(--warning)_16%,transparent)]",
      )}
    />
  );
}

function MetricPill({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border/80 bg-background/70 px-4 py-3 shadow-sm backdrop-blur">
      <p className="font-mono text-[10px] font-medium tracking-[0.18em] text-muted-foreground">
        {label}
      </p>
      <p className="mt-1 font-mono text-lg font-semibold tabular-nums">{value}</p>
    </div>
  );
}

function MonitoringConsolePreview() {
  return (
    <figure className="relative mx-auto w-full max-w-xl lg:max-w-none">
      <figcaption className="sr-only">Live monitoring console preview</figcaption>
      <div
        aria-hidden
        className="pointer-events-none absolute -inset-6 -z-10 rounded-[2rem] bg-primary/15 blur-3xl dark:bg-primary/20"
      />

      <div
        dir="ltr"
        className="overflow-hidden rounded-2xl border border-border/80 bg-card/95 text-start shadow-2xl shadow-foreground/10 backdrop-blur-xl dark:border-primary/15 dark:bg-card/80 dark:shadow-primary/10"
      >
        <div className="border-b border-border/70 p-4 sm:p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="font-mono text-[10px] font-medium tracking-[0.2em] text-muted-foreground">
                MONITORING CONTROL ROOM
              </p>
              <h2 className="mt-1 text-lg font-semibold tracking-tight">Production endpoints</h2>
            </div>
            <span className="inline-flex items-center gap-2 rounded-full border border-success/25 bg-success/10 px-3 py-1 font-mono text-[10px] font-semibold tracking-wider text-success">
              <span className="relative flex size-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-60" />
                <span className="relative inline-flex size-2 rounded-full bg-success" />
              </span>
              LIVE PROBES
            </span>
          </div>

          <div className="mt-5 grid grid-cols-3 gap-2">
            <MetricPill label="UP" value="4" />
            <MetricPill label="DEGRADED" value="1" />
            <MetricPill label="DOWN" value="1" />
          </div>
        </div>

        <div className="grid min-w-0 grid-cols-1 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]">
          <div className="border-b border-border/70 p-4 lg:border-b-0 lg:border-e">
            <div className="mb-3 flex items-center justify-between">
              <span className="font-mono text-[10px] font-medium tracking-[0.18em] text-muted-foreground">
                CHECKS
              </span>
              <span className="font-mono text-[10px] text-muted-foreground">HTTP · TLS · DNS · SMTP · NTP</span>
            </div>

            <ul className="space-y-2">
              {SIGNAL_ROWS.map((row) => {
                const Icon = row.icon;
                return (
                  <li
                    key={row.host}
                    className="flex min-w-0 items-center gap-3 rounded-xl border border-border/70 bg-background/60 px-3 py-2.5"
                  >
                    <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-mono text-xs font-medium">{row.host}</p>
                      <p className="mt-0.5 font-mono text-[10px] tracking-wider text-muted-foreground">
                        {row.kind}
                      </p>
                    </div>
                    <span className="w-16 shrink-0 text-end font-mono text-xs tabular-nums text-muted-foreground">
                      {row.latency}
                    </span>
                    <StatusDot status={row.status} />
                  </li>
                );
              })}
            </ul>
          </div>

          <div className="space-y-4 p-4">
            <div className="rounded-xl border border-border/70 bg-background/60 p-3.5">
              <div className="mb-3 flex items-center justify-between">
                <span className="font-mono text-[10px] font-medium tracking-[0.18em] text-muted-foreground">
                  LATENCY
                </span>
                <span className="font-mono text-[10px] text-primary">P95 WATCH</span>
              </div>
              <div className="flex h-24 items-end gap-1">
                {LATENCY_BARS.map((height, index) => (
                  <span
                    key={index}
                    style={{ height: `${height}%` }}
                    className={cn(
                      "min-w-0 flex-1 rounded-sm",
                      height > 75 ? "bg-warning" : "bg-primary/55",
                    )}
                  />
                ))}
              </div>
            </div>

            <div className="rounded-xl border border-border/70 bg-background/60 p-3.5">
              <div className="mb-3 flex items-center gap-2">
                <BellRing className="size-3.5 text-destructive" aria-hidden />
                <span className="font-mono text-[10px] font-medium tracking-[0.18em] text-muted-foreground">
                  INCIDENT SIGNAL
                </span>
              </div>
              <ul className="space-y-2">
                {INCIDENTS.map((event) => (
                  <li key={`${event.time}-${event.title}`} className="flex items-start gap-2">
                    <span className="mt-1.5"><StatusDot status={event.tone as "up" | "down" | "warn"} /></span>
                    <div className="min-w-0">
                      <p className="font-mono text-[10px] text-muted-foreground">{event.time}</p>
                      <p className="text-xs font-medium">{event.title}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </div>
    </figure>
  );
}

export default async function LandingPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);

  const t = await getTranslations({ locale, namespace: "landing" });
  const tTypes = await getTranslations({ locale, namespace: "types" });

  return (
    <div className="overflow-x-clip">
      <section className="relative overflow-hidden border-b border-border/70">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-grid-faint [mask-image:radial-gradient(ellipse_75%_62%_at_50%_0%,black,transparent)]"
        />
        <div
          aria-hidden
          className="pointer-events-none absolute -top-56 end-0 size-[34rem] rounded-full bg-primary/10 blur-3xl dark:bg-primary/15"
        />
        <div
          aria-hidden
          className="pointer-events-none absolute bottom-10 start-0 size-80 rounded-full bg-success/8 blur-3xl dark:bg-success/10"
        />

        <div className="relative mx-auto grid w-full max-w-7xl items-center gap-12 px-4 py-16 sm:py-20 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] lg:py-24">
          <div className="min-w-0">
            <p className="inline-flex items-center gap-2 rounded-full border border-border bg-card/85 px-3 py-1 text-xs font-medium text-muted-foreground shadow-sm backdrop-blur">
              <Activity className="size-3.5 text-primary" aria-hidden />
              {t("badge")}
            </p>

            <h1 className="mt-6 max-w-3xl text-balance font-display text-4xl font-bold leading-[1.06] tracking-tight sm:text-5xl xl:text-6xl">
              {t("heroTitle")}
            </h1>

            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              {t("heroSubtitle")}
            </p>

            <div className="mt-8 flex flex-wrap items-center gap-3">
              <Button asChild size="lg" className="h-11 px-6 text-base">
                <Link href="/app/monitors/new">{t("ctaPrimary")}</Link>
              </Button>
              <Button asChild size="lg" variant="outline" className="h-11 px-6 text-base">
                <Link href="/app/dashboard">{t("ctaSecondary")}</Link>
              </Button>
            </div>

            <div className="mt-7 grid max-w-xl grid-cols-1 gap-2 sm:grid-cols-3">
              <div className="rounded-xl border border-border/70 bg-card/70 px-3 py-2.5">
                <p className="font-mono text-[10px] tracking-[0.18em] text-muted-foreground">PROBES</p>
                <p className="mt-1 text-sm font-semibold">HTTP · TCP · DNS</p>
              </div>
              <div className="rounded-xl border border-border/70 bg-card/70 px-3 py-2.5">
                <p className="font-mono text-[10px] tracking-[0.18em] text-muted-foreground">SIGNALS</p>
                <p className="mt-1 text-sm font-semibold">TLS · SMTP · NTP</p>
              </div>
              <div className="rounded-xl border border-border/70 bg-card/70 px-3 py-2.5">
                <p className="font-mono text-[10px] tracking-[0.18em] text-muted-foreground">DELIVERY</p>
                <p className="mt-1 text-sm font-semibold">Live SSE</p>
              </div>
            </div>

            <p className="mt-5 text-sm text-muted-foreground">{t("heroNote")}</p>
          </div>

          <MonitoringConsolePreview />
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 py-20 lg:py-28">
        <div className="grid gap-10 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
          <SectionHeader
            eyebrow={t("typesEyebrow")}
            title={t("typesTitle")}
            subtitle={t("typesSubtitle")}
          />

          <div className="grid grid-cols-1 gap-px overflow-hidden rounded-2xl border border-border bg-border sm:grid-cols-2 xl:grid-cols-4">
            {MONITOR_TYPES.map((monitorType) => {
              const Icon = MONITOR_TYPE_ICONS[monitorType];
              return (
                <div
                  key={monitorType}
                  className="group flex min-w-0 flex-col gap-4 bg-card p-5 transition-colors hover:bg-accent/45"
                >
                  <span className="grid size-10 place-items-center rounded-xl border border-border/70 bg-background text-primary transition-colors group-hover:border-primary/40">
                    <Icon className="size-4.5" aria-hidden />
                  </span>
                  <div className="min-w-0">
                    <h3 className="text-sm font-semibold">{tTypes(monitorType)}</h3>
                    <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                      {t(`typeDesc.${monitorType}`)}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <SectionHeader
          eyebrow={t("bentoEyebrow")}
          title={t("bentoTitle")}
          subtitle={t("bentoSubtitle")}
        />

        <div className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <BentoTile
            title={t("liveTitle")}
            body={t("liveBody")}
            className="relative overflow-hidden lg:col-span-2"
          >
            <div
              aria-hidden
              className="absolute end-6 top-6 size-24 rounded-full bg-success/10 blur-2xl"
            />
            <div
              dir="ltr"
              className="w-full overflow-hidden rounded-xl border border-border/70 bg-muted/40 p-3.5 text-start font-mono text-[11px] leading-6"
            >
              {COMMANDS.map((line) => (
                <p key={line} className="whitespace-pre text-muted-foreground">
                  <span className={line.includes("failed") ? "text-destructive" : "text-success"}>
                    {line.includes("failed") ? "×" : "✓"}
                  </span>{" "}
                  {line}
                </p>
              ))}
            </div>
          </BentoTile>

          <BentoTile title={t("securityTitle")} body={t("securityBody")}>
            <div dir="ltr" className="w-full rounded-xl border border-border/70 bg-muted/40 p-3.5 text-start font-mono text-[11px] leading-6">
              {["10.0.0.0/8", "169.254.169.254", "fd00::/8"].map((target) => (
                <p key={target} className="flex items-center justify-between gap-4 text-muted-foreground">
                  <span>{target}</span>
                  <span className="text-destructive">blocked</span>
                </p>
              ))}
            </div>
          </BentoTile>

          <BentoTile title={t("locationsTitle")} body={t("locationsBody")}>
            <div dir="ltr" className="grid w-full grid-cols-3 gap-2">
              {["FRA", "AMS", "THR"].map((location) => (
                <span
                  key={location}
                  className="inline-flex items-center justify-center gap-1.5 rounded-xl border border-border/70 bg-background px-2.5 py-2 font-mono text-[11px] text-muted-foreground"
                >
                  <span className="size-1.5 rounded-full bg-success" />
                  {location}
                </span>
              ))}
            </div>
          </BentoTile>

          <BentoTile title={t("historyTitle")} body={t("historyBody")}>
            <div dir="ltr" className="flex h-20 w-full items-end gap-1 rounded-xl border border-border/70 bg-muted/40 p-3">
              {LATENCY_BARS.map((height, index) => (
                <span
                  key={index}
                  style={{ height: `${height}%` }}
                  className={cn("min-w-0 flex-1 rounded-sm", height > 75 ? "bg-warning/70" : "bg-primary/50")}
                />
              ))}
            </div>
          </BentoTile>

          <BentoTile title={t("scheduleTitle")} body={t("scheduleBody")}>
            <div dir="ltr" className="flex w-full flex-wrap gap-2">
              {["interval 60s", "timeout 5s", "retries 3"].map((chip) => (
                <span
                  key={chip}
                  className="rounded-xl border border-border/70 bg-background px-3 py-1.5 font-mono text-[11px] text-muted-foreground"
                >
                  {chip}
                </span>
              ))}
            </div>
          </BentoTile>
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <div className="rounded-3xl border border-border bg-card p-6 sm:p-8 lg:p-10">
          <SectionHeader
            eyebrow={t("howEyebrow")}
            title={t("howTitle")}
            subtitle={t("howSubtitle")}
          />

          <div className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-3">
            {(
              [
                { number: "01", title: t("how1Title"), body: t("how1Body") },
                { number: "02", title: t("how2Title"), body: t("how2Body") },
                { number: "03", title: t("how3Title"), body: t("how3Body") },
              ] as const
            ).map((step) => (
              <div key={step.number} className="rounded-2xl border border-border/70 bg-background/60 p-5">
                <span className="grid size-8 place-items-center rounded-xl bg-primary/10 font-mono text-xs font-semibold text-primary" dir="ltr">
                  {step.number}
                </span>
                <h3 className="mt-5 font-semibold">{step.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{step.body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="relative overflow-hidden border-t border-border/70">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-grid-faint [mask-image:radial-gradient(ellipse_55%_70%_at_50%_100%,black,transparent)]"
        />
        <div className="relative mx-auto flex w-full max-w-7xl flex-col items-center gap-6 px-4 py-20 text-center lg:py-28">
          <h2 className="max-w-2xl text-balance text-3xl font-bold tracking-tight lg:text-4xl">
            {t("ctaTitle")}
          </h2>
          <p className="max-w-md text-pretty text-muted-foreground">{t("ctaSubtitle")}</p>
          <div className="mt-2 flex flex-wrap items-center justify-center gap-3">
            <Button asChild size="lg" className="h-11 px-6 text-base">
              <Link href="/app/monitors/new">{t("ctaPrimary")}</Link>
            </Button>
            <Button asChild size="lg" variant="outline" className="h-11 px-6 text-base">
              <Link href="/app/dashboard">{t("ctaSecondary")}</Link>
            </Button>
          </div>
        </div>
      </section>
    </div>
  );
}
