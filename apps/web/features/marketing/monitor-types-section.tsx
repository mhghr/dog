"use client";

import { useTranslations } from "next-intl";

import { AnimatedGroup } from "@/shared/ui/motion/animated-group";
import { TextEffect } from "@/shared/ui/motion/text-effect";
import {
  Broadcast,
  CalendarCheck,
  Clock,
  EnvelopeSimple,
  Globe,
  PlugsConnected,
  ShieldCheck,
  TreeStructure,
} from "@/shared/ui/icons";

// Lightweight: inline static icons for the landing page.
// The heavy plugin registry lives in plugins/monitoring and is NOT imported here.
const LANDING_MONITOR_TYPES = [
  { type: "http",             icon: Globe,           label: "http" },
  { type: "tcp",              icon: PlugsConnected,  label: "tcp" },
  { type: "dns",              icon: TreeStructure,   label: "dns" },
  { type: "ping",             icon: Broadcast,       label: "ping" },
  { type: "tls",              icon: ShieldCheck,     label: "tls" },
  { type: "domain_expiration",icon: CalendarCheck,   label: "domain_expiration" },
  { type: "smtp",             icon: EnvelopeSimple,  label: "smtp" },
  { type: "ntp",              icon: Clock,           label: "ntp" },
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

export default function MonitorTypesSection() {
  const t = useTranslations("landing");
  const tTypes = useTranslations("types");

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
                {t("typesEyebrow")}
              </span>
            </AnimatedGroup>

            <TextEffect
              preset="fade-in-blur"
              speedSegment={0.3}
              as="h2"
              className="mx-auto mt-6 max-w-3xl text-balance text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl"
            >
              {t("typesTitle")}
            </TextEffect>

            <TextEffect
              per="line"
              preset="fade-in-blur"
              speedSegment={0.3}
              delay={0.3}
              as="p"
              className="mx-auto mt-5 max-w-2xl text-balance text-lg"
            >
              {t("typesSubtitle")}
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
              <div className="grid grid-cols-1 gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-2 xl:grid-cols-4">
                {LANDING_MONITOR_TYPES.map((item) => {
                  const Icon = item.icon;
                  return (
                    <div
                      key={item.type}
                      className="group flex min-w-0 flex-col gap-4 bg-card p-5 transition-colors hover:bg-accent/45"
                    >
                      <span className="grid size-10 place-items-center rounded-xl border border-border/70 bg-background text-primary transition-colors group-hover:border-primary/40">
                        <Icon className="size-4.5" aria-hidden />
                      </span>
                      <div className="min-w-0">
                        <h3 className="text-sm font-semibold">{tTypes(item.type)}</h3>
                        <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                          {t(`typeDesc.${item.type}`)}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </AnimatedGroup>
      </div>
    </section>
  );
}
