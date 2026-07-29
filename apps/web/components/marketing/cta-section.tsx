"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";
import { AnimatedGroup } from "@/components/motion-primitives/animated-group";
import { TextEffect } from "@/components/motion-primitives/text-effect";

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

export default function CTASection() {
  const t = useTranslations("landing");

  return (
    <section className="relative overflow-hidden border-t border-border/70">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-grid-faint [mask-image:radial-gradient(ellipse_55%_70%_at_50%_100%,black,transparent)]"
      />
      <div className="relative mx-auto flex w-full max-w-7xl flex-col items-center gap-6 px-4 py-20 text-center lg:py-28">
        <TextEffect
          preset="fade-in-blur"
          speedSegment={0.3}
          as="h2"
          className="max-w-2xl text-balance text-3xl font-bold tracking-tight lg:text-4xl"
        >
          {t("ctaTitle")}
        </TextEffect>

        <TextEffect
          per="line"
          preset="fade-in-blur"
          speedSegment={0.3}
          delay={0.3}
          as="p"
          className="max-w-md text-pretty text-muted-foreground"
        >
          {t("ctaSubtitle")}
        </TextEffect>

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
          className="mt-2 flex flex-wrap items-center justify-center gap-3"
        >
          <Button asChild size="lg" className="h-11 px-6 text-base">
            <Link href="/app/nodes/new">{t("ctaPrimary")}</Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="h-11 px-6 text-base">
            <Link href="/app/dashboard">{t("ctaSecondary")}</Link>
          </Button>
        </AnimatedGroup>
      </div>
    </section>
  );
}
