import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketing" });

  return {
    title: t("privacy.title"),
    alternates: {
      canonical: `/${locale}/privacy`,
      languages: { en: "/en/privacy", fa: "/fa/privacy" },
    },
  };
}

export default async function PrivacyPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);

  const t = await getTranslations({ locale, namespace: "marketing" });

  return (
    <div>
      <section className="mx-auto w-full max-w-2xl px-4 py-20 lg:py-28">
        <h1 className="text-balance text-2xl font-bold tracking-tight sm:text-3xl">
          {t("privacy.title")}
        </h1>
        <p className="mt-4 text-sm text-muted-foreground">
          {t("privacy.lastUpdated")}
        </p>

        <div className="mt-10 space-y-8 text-pretty leading-relaxed text-muted-foreground sm:text-base">
          {Array.from({ length: 7 }, (_, i) => (
            <section key={i}>
              <h2 className="mb-3 text-lg font-semibold text-foreground">
                {t(`privacy.section${i + 1}Title`)}
              </h2>
              <p>{t(`privacy.section${i + 1}Body`)}</p>
            </section>
          ))}
        </div>
      </section>
    </div>
  );
}
