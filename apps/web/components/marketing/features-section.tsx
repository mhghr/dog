"use client";

import { useTranslations } from "next-intl";

import { AnimatedGroup } from "@/components/motion-primitives/animated-group";
import { TextEffect } from "@/components/motion-primitives/text-effect";
import { cn } from "@/lib/utils";

const LATENCY_BARS = [36, 44, 41, 58, 52, 69, 47, 55, 82, 49, 43, 61, 39, 46, 73, 51];

const COMMANDS = [
  "http  api.production.local  200  142ms",
  "tls   app.company.com       expires in 61d",
  "dns   company.com           A resolved",
  "smtp  mail.company.com      starttls failed",
];

const transitionVariants = {
  item: {
    hidden: {
      opacity: 0,
      filter: "blur(12px)",
      y: 12,
    },
    visible: {
      opacity: 1,
      filter: "blur(0px)",
      y: 0,
      transition: {
        type: "spring",
        duration: 1.5,
      },
    },
  },
};

export default function FeaturesSection() {
  const t = useTranslations("landing");

  return (
    <section className="relative overflow-hidden border-b border-border/70">
      <div
        aria-hidden
        className="pointer-events-none absolute -inset-1 -z-10 bg-[radial-gradient(60%_60%_at_50%_35%,hsla(0,0%,85%,.05)_0,transparent_100%)]"
      />

      <div className="relative pt-24 md:pt-32">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 -z-10 size-full [background:radial-gradient(125%_125%_at_50%_100%,transparent_0%,var(--color-background)_75%)]"
        />

        <div className="mx-auto max-w-7xl px-6">
          <div className="text-center">
            <AnimatedGroup variants={transitionVariants}>
              <span className="inline-flex items-center gap-2 rounded-full border border-border bg-card/85 px-3 py-1 text-xs font-medium text-muted-foreground shadow-sm backdrop-blur">
                {t("bentoEyebrow")}
              </span>
            </AnimatedGroup>

            <TextEffect
              preset="fade-in-blur"
              speedSegment={0.3}
              as="h2"
              className="mx-auto mt-6 max-w-3xl text-balance text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl"
            >
              {t("bentoTitle")}
            </TextEffect>

            <TextEffect
              per="line"
              preset="fade-in-blur"
              speedSegment={0.3}
              delay={0.3}
              as="p"
              className="mx-auto mt-5 max-w-2xl text-balance text-lg"
            >
              {t("bentoSubtitle")}
            </TextEffect>
          </div>
        </div>

        <AnimatedGroup
          variants={{
            container: {
              visible: {
                transition: {
                  staggerChildren: 0.06,
                  delayChildren: 0.5,
                },
              },
            },
            ...transitionVariants,
          }}
        >
          <div className="mask-b-from-55% relative mt-10 overflow-hidden px-4 sm:mt-14 md:mt-20">
            <div className="inset-shadow-2xs ring-background dark:inset-shadow-white/20 bg-background relative mx-auto max-w-5xl overflow-hidden rounded-2xl border p-4 shadow-lg shadow-zinc-950/15 ring-1">
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
                <div className="relative overflow-hidden rounded-xl border border-border bg-card p-5 lg:col-span-2">
                  <div
                    aria-hidden
                    className="pointer-events-none absolute end-6 top-6 size-24 rounded-full bg-success/10 blur-2xl"
                  />
                  <h3 className="font-semibold">{t("liveTitle")}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {t("liveBody")}
                  </p>
                  <div
                    dir="ltr"
                    className="mt-5 w-full overflow-hidden rounded-xl border border-border/70 bg-muted/40 p-3.5 text-start font-mono text-[11px] leading-6"
                  >
                    {COMMANDS.map((line) => (
                      <p key={line} className="whitespace-pre text-muted-foreground">
                        <span className={line.includes("failed") ? "text-destructive" : "text-success"}>
                          {line.includes("failed") ? "x" : "v"}
                        </span>{" "}
                        {line}
                      </p>
                    ))}
                  </div>
                </div>

                <div className="flex flex-col rounded-xl border border-border bg-card p-5">
                  <h3 className="font-semibold">{t("securityTitle")}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {t("securityBody")}
                  </p>
                  <div dir="ltr" className="mt-5 w-full rounded-xl border border-border/70 bg-muted/40 p-3.5 text-start font-mono text-[11px] leading-6">
                    {["10.0.0.0/8", "169.254.169.254", "fd00::/8"].map((target) => (
                      <p key={target} className="flex items-center justify-between gap-4 text-muted-foreground">
                        <span>{target}</span>
                        <span className="text-destructive">blocked</span>
                      </p>
                    ))}
                  </div>
                </div>

                <div className="flex flex-col rounded-xl border border-border bg-card p-5">
                  <h3 className="font-semibold">{t("locationsTitle")}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {t("locationsBody")}
                  </p>
                  <div dir="ltr" className="mt-5 grid w-full grid-cols-3 gap-2">
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
                </div>

                <div className="flex flex-col rounded-xl border border-border bg-card p-5">
                  <h3 className="font-semibold">{t("historyTitle")}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {t("historyBody")}
                  </p>
                  <div dir="ltr" className="mt-5 flex h-20 w-full items-end gap-1 rounded-xl border border-border/70 bg-muted/40 p-3">
                    {LATENCY_BARS.map((height, index) => (
                      <span
                        key={index}
                        style={{ height: `${height}%` }}
                        className={cn("min-w-0 flex-1 rounded-sm", height > 75 ? "bg-warning/70" : "bg-primary/50")}
                      />
                    ))}
                  </div>
                </div>

                <div className="flex flex-col rounded-xl border border-border bg-card p-5">
                  <h3 className="font-semibold">{t("scheduleTitle")}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {t("scheduleBody")}
                  </p>
                  <div dir="ltr" className="mt-5 flex w-full flex-wrap gap-2">
                    {["interval 60s", "timeout 5s", "retries 3"].map((chip) => (
                      <span
                        key={chip}
                        className="rounded-xl border border-border/70 bg-background px-3 py-1.5 font-mono text-[11px] text-muted-foreground"
                      >
                        {chip}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </AnimatedGroup>
      </div>
    </section>
  );
}
