import type { Metadata } from "next";
import { Check } from "lucide-react";
import { getTranslations, setRequestLocale } from "next-intl/server";

import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";
import { cn } from "@/lib/utils";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketing" });

  return {
    title: t("pricing.title"),
    description: t("pricing.subtitle"),
    alternates: {
      canonical: `/${locale}/pricing`,
      languages: { en: "/en/pricing", fa: "/fa/pricing" },
    },
  };
}

const PLAN_KEYS = ["free", "starter", "pro", "business"] as const;

export default async function PricingPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);

  const t = await getTranslations({ locale, namespace: "marketing" });

  return (
    <div>
      <section className="relative overflow-hidden">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-grid-faint [mask-image:radial-gradient(ellipse_75%_65%_at_50%_0%,black,transparent)]"
        />
        <div className="relative mx-auto w-full max-w-7xl px-4 pb-20 pt-16 lg:pb-28 lg:pt-24">
          <div className="max-w-2xl">
            <p className="text-sm font-semibold text-primary">
              {t("pricing.title")}
            </p>
            <h1 className="mt-2 text-balance text-3xl font-bold tracking-tight sm:text-4xl">
              {t("pricing.title")}
            </h1>
            <p className="mt-4 text-pretty leading-relaxed text-muted-foreground sm:text-lg">
              {t("pricing.subtitle")}
            </p>
          </div>

          <div className="mt-14 grid grid-cols-1 gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-4">
            {PLAN_KEYS.map((key, i) => (
              <div
                key={key}
                className={cn(
                  "flex flex-col bg-card p-6",
                  i === 1 && "sm:scale-105 sm:shadow-lg sm:shadow-foreground/5 sm:z-10 sm:rounded-xl",
                )}
              >
                <p className="text-xs font-semibold tracking-wide text-primary uppercase">
                  {t(`pricing.plans.${key}.name`)}
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {t(`pricing.plans.${key}.description`)}
                </p>

                <div className="mt-6 flex items-baseline gap-1">
                  <span className="text-3xl font-bold tracking-tight">
                    {t("pricing.comingSoon")}
                  </span>
                </div>

                <div className="mt-6 space-y-3 border-t border-border/70 pt-5">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">
                      {t(`pricing.plans.${key}.retentionLabel`)}
                    </span>
                    <span className="font-mono text-sm font-semibold tabular-nums">
                      {t(`pricing.plans.${key}.retention`)}
                    </span>
                  </div>
                </div>

                <div className="mt-auto pt-6">
                  <Button
                    asChild
                    variant={i === 1 ? "default" : "outline"}
                    className="w-full"
                  >
                    <Link href="/app/monitors/new">
                      {t("pricing.ctaFree")}
                    </Link>
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <h2 className="text-lg font-bold tracking-tight">
          {t("pricing.featureLabel")}
        </h2>

        <div className="mt-6 grid gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-2">
          {t.raw("pricing.features").map((feature: string, i: number) => (
            <div
              key={i}
              className="flex items-start gap-3 bg-card px-5 py-4"
            >
              <Check className="mt-0.5 size-4 shrink-0 text-primary" aria-hidden />
              <span className="text-sm leading-relaxed">{feature}</span>
            </div>
          ))}
        </div>

        <p className="mt-10 rounded-xl border border-border bg-muted/50 px-5 py-4 text-center text-sm leading-relaxed text-muted-foreground">
          {t("pricing.billingNote")}
        </p>
      </section>
    </div>
  );
}
