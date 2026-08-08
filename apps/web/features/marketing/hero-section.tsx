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

export default function HeroSection() {
  const t = useTranslations("landing");

  return (
    <>
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 isolate hidden opacity-65 contain-strict lg:block"
      >
        <div className="absolute left-0 top-0 h-80 w-140 -translate-y-[87.5%] -rotate-45 rounded-full bg-[radial-gradient(68.54%_68.72%_at_55.02%_31.46%,hsla(0,0%,85%,.08)_0,hsla(0,0%,55%,.02)_50%,hsla(0,0%,45%,0)_80%)]" />
        <div className="absolute left-0 top-0 h-80 w-60 -rotate-45 rounded-full bg-[radial-gradient(50%_50%_at_50%_50%,hsla(0,0%,85%,.06)_0,hsla(0,0%,45%,.02)_80%,transparent_100%)] [translate:5%_-50%]" />
        <div className="absolute left-0 top-0 h-80 w-60 -rotate-45 -translate-y-[87.5%] rounded-full bg-[radial-gradient(50%_50%_at_50%_50%,hsla(0,0%,85%,.04)_0,hsla(0,0%,45%,.02)_80%,transparent_100%)]" />
      </div>

      <section className="relative overflow-hidden border-b border-border/70">
        <div className="relative pt-24 md:pt-36">
          <div
            aria-hidden
            className="pointer-events-none absolute inset-0 -z-10 size-full [background:radial-gradient(125%_125%_at_50%_100%,transparent_0%,var(--color-background)_75%)]"
          />

          <div className="mx-auto max-w-7xl px-6">
            <div className="text-center sm:mx-auto lg:mr-auto lg:mt-0">
              <TextEffect
                preset="fade-in-blur"
                speedSegment={0.3}
                as="h1"
                className="mx-auto max-w-4xl text-balance text-5xl font-semibold tracking-tight md:text-7xl xl:text-[5.25rem] [font-family:var(--font-plasma)]"
              >
                {t("heroTitle")}
              </TextEffect>

              <TextEffect
                per="line"
                preset="fade-in-blur"
                speedSegment={0.3}
                delay={0.5}
                as="p"
                className="mx-auto mt-8 max-w-2xl text-balance text-lg"
              >
                {t("heroSubtitle")}
              </TextEffect>
            </div>
          </div>

          <AnimatedGroup
            variants={{
              container: {
                visible: {
                  transition: {
                    staggerChildren: 0.05,
                    delayChildren: 0.75,
                  },
                },
              },
              ...transitionVariants,
            }}
          >
            <div className="mask-b-from-55% relative -mr-56 mt-8 overflow-hidden px-2 sm:mr-0 sm:mt-12 md:mt-20">
              <div className="inset-shadow-2xs ring-background dark:inset-shadow-white/20 bg-background relative mx-auto max-w-6xl overflow-hidden rounded-2xl border p-4 shadow-lg shadow-zinc-950/15 ring-1">
                <div className="bg-background relative flex min-h-[360px] items-center justify-center rounded-2xl">
                  <span className="font-mono text-sm text-muted-foreground/60">
                    Product Screenshot
                  </span>
                </div>
              </div>
            </div>
          </AnimatedGroup>
        </div>
      </section>

      <section className="bg-background pb-16 pt-16 md:pb-32">
        <div className="mx-auto max-w-5xl px-6">
          <AnimatedGroup
            variants={{
              container: {
                visible: {
                  transition: {
                    staggerChildren: 0.05,
                    delayChildren: 0.3,
                  },
                },
              },
              ...transitionVariants,
            }}
          >
            <p className="text-center text-sm font-medium text-muted-foreground">
              {t("heroNote")}
            </p>
          </AnimatedGroup>

          <AnimatedGroup
            variants={{
              container: {
                visible: {
                  transition: {
                    staggerChildren: 0.05,
                    delayChildren: 0.5,
                  },
                },
              },
              ...transitionVariants,
            }}
            className="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6"
          >
            {["HTTP", "TLS", "DNS", "SMTP", "NTP", "TCP"].map((label) => (
              <div
                key={label}
                className="flex items-center justify-center rounded-xl border border-border/70 bg-card/60 px-4 py-3"
              >
                <span className="font-mono text-sm font-semibold text-muted-foreground">
                  {label}
                </span>
              </div>
            ))}
          </AnimatedGroup>
        </div>
      </section>
    </>
  );
}
