"use client";

import { useTranslations } from "next-intl";

import { AnimatedGroup } from "@/shared/ui/motion/animated-group";
import { TextEffect } from "@/shared/ui/motion/text-effect";

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

export default function HowItWorksSection() {
  const t = useTranslations("landing");

  const steps = [
    { number: "01", title: t("how1Title"), body: t("how1Body") },
    { number: "02", title: t("how2Title"), body: t("how2Body") },
    { number: "03", title: t("how3Title"), body: t("how3Body") },
  ] as const;

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
                {t("howEyebrow")}
              </span>
            </AnimatedGroup>

            <TextEffect
              preset="fade-in-blur"
              speedSegment={0.3}
              as="h2"
              className="mx-auto mt-6 max-w-3xl text-balance text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl"
            >
              {t("howTitle")}
            </TextEffect>

            <TextEffect
              per="line"
              preset="fade-in-blur"
              speedSegment={0.3}
              delay={0.3}
              as="p"
              className="mx-auto mt-5 max-w-2xl text-balance text-lg"
            >
              {t("howSubtitle")}
            </TextEffect>
          </div>
        </div>

        <AnimatedGroup
          variants={{
            container: {
              visible: {
                transition: {
                  staggerChildren: 0.08,
                  delayChildren: 0.5,
                },
              },
            },
            ...transitionVariants,
          }}
        >
          <div className="mask-b-from-55% relative mt-10 overflow-hidden px-4 sm:mt-14 md:mt-20">
            <div className="mx-auto max-w-5xl">
              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                {steps.map((step) => (
                  <div
                    key={step.number}
                    className="inset-shadow-2xs ring-background dark:inset-shadow-white/20 bg-background rounded-2xl border p-6 shadow-lg shadow-zinc-950/15 ring-1"
                  >
                    <span
                      className="grid size-10 place-items-center rounded-xl bg-primary/10 font-mono text-sm font-semibold text-primary"
                      dir="ltr"
                    >
                      {step.number}
                    </span>
                    <h3 className="mt-5 font-semibold">{step.title}</h3>
                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                      {step.body}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </AnimatedGroup>
      </div>
    </section>
  );
}
