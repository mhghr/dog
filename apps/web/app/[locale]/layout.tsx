import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { hasLocale, NextIntlClientProvider } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";

import { AppProviders } from "@/platform/providers/providers";
import { routing } from "@/i18n/routing";
import { bakh, estedad, inter, plasma } from "@/shared/ui/fonts";

import "../globals.css";

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "common" });

  return {
    title: {
      default: t("appName"),
      template: `%s | ${t("appName")}`,
    },
    description:
      locale === "fa"
        ? "پلتفرم مانیتورینگ Agentless برای HTTP، Ping، TCP، DNS، TLS، دامنه، SMTP و NTP"
        : "Agentless monitoring platform for HTTP, Ping, TCP, DNS, TLS, domains, SMTP and NTP",
  };
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }

  setRequestLocale(locale);

  return (
    <html
      lang={locale}
      dir={locale === "fa" ? "rtl" : "ltr"}
      suppressHydrationWarning
      className={`${estedad.variable} ${bakh.variable} ${plasma.variable} ${inter.variable}`}
    >
      <body className="min-h-screen bg-background font-sans text-foreground antialiased">
        <NextIntlClientProvider>
          <AppProviders>{children}</AppProviders>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
