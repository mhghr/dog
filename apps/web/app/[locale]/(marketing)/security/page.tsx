import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";

const BLOCKED_RANGES = [
  "10.0.0.0/8",
  "172.16.0.0/12",
  "192.168.0.0/16",
  "169.254.0.0/16",
  "127.0.0.0/8",
  "0.0.0.0/8",
  "100.64.0.0/10",
  "198.18.0.0/15",
  "224.0.0.0/4",
  "240.0.0.0/4",
  "fd00::/8",
  "::1/128",
  "fe80::/10",
];

function SecuritySection({
  eyebrow,
  title,
  children,
}: {
  eyebrow: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 sm:p-8">
      <p className="text-sm font-semibold text-primary">{eyebrow}</p>
      <h2 className="mt-2 text-pretty text-xl font-bold tracking-tight sm:text-2xl">
        {title}
      </h2>
      <div className="mt-4 space-y-3 text-pretty leading-relaxed text-muted-foreground">
        {children}
      </div>
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
    title: t("security.title"),
    description: t("security.subtitle"),
    alternates: {
      canonical: `/${locale}/security`,
      languages: { en: "/en/security", fa: "/fa/security" },
    },
  };
}

export default async function SecurityPage({
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
              {t("security.title")}
            </p>
            <h1 className="mt-2 text-balance text-3xl font-bold tracking-tight sm:text-4xl">
              {t("security.title")}
            </h1>
            <p className="mt-4 text-pretty leading-relaxed text-muted-foreground sm:text-lg">
              {t("security.subtitle")}
            </p>
          </div>
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 pb-20 lg:pb-28">
        <div className="space-y-8">
          <SecuritySection
            eyebrow={t("security.ssrfEyebrow")}
            title={t("security.ssrfTitle")}
          >
            <p>{t("security.ssrfBody1")}</p>
            <p>{t("security.ssrfBody2")}</p>

            <div className="mt-5">
              <p className="mb-3 font-mono text-xs font-medium tracking-wider text-primary">
                {t("security.blockedRangesLabel")}
              </p>
              <div
                dir="ltr"
                className="grid gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-3"
              >
                {BLOCKED_RANGES.map((range) => (
                  <div
                    key={range}
                    className="flex items-center justify-between bg-card px-4 py-3"
                  >
                    <span className="font-mono text-caption-lg">{range}</span>
                    <span className="font-mono text-caption font-medium text-destructive">
                      BLOCKED
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </SecuritySection>

          <SecuritySection
            eyebrow={t("security.validationEyebrow")}
            title={t("security.validationTitle")}
          >
            <p>{t("security.validationBody1")}</p>
            <p>{t("security.validationBody2")}</p>
          </SecuritySection>

          <SecuritySection
            eyebrow={t("security.authEyebrow")}
            title={t("security.authTitle")}
          >
            <p>{t("security.authBody1")}</p>
            <p>{t("security.authBody2")}</p>
          </SecuritySection>

          <SecuritySection
            eyebrow={t("security.workerEyebrow")}
            title={t("security.workerTitle")}
          >
            <p>{t("security.workerBody")}</p>
          </SecuritySection>

          <SecuritySection
            eyebrow={t("security.redactionEyebrow")}
            title={t("security.redactionTitle")}
          >
            <p>{t("security.redactionBody")}</p>
          </SecuritySection>

          <SecuritySection
            eyebrow={t("security.rateLimitEyebrow")}
            title={t("security.rateLimitTitle")}
          >
            <p>{t("security.rateLimitBody")}</p>
          </SecuritySection>
        </div>
      </section>
    </div>
  );
}
