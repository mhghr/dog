import { getTranslations, setRequestLocale } from "next-intl/server";

import { AuthButton } from "@/components/layout/auth-button";
import { BrandMark } from "@/components/layout/brand-mark";
import { LanguageSwitcher } from "@/components/layout/language-switcher";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { Link } from "@/i18n/navigation";

export default async function MarketingLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);

  const t = await getTranslations({ locale, namespace: "navigation" });
  const tCommon = await getTranslations({ locale, namespace: "common" });
  const tLanding = await getTranslations({ locale, namespace: "landing" });

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-20 border-b border-border/60 bg-background/70 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60 dark:border-primary/5">
        <div className="mx-auto flex h-14 w-full max-w-7xl items-center gap-4 px-4">
          <Link href="/" className="flex items-center gap-2.5">
            <BrandMark />
            <span className="text-sm font-semibold tracking-tight">
              {tCommon("appName")}
            </span>
          </Link>

          <nav className="hidden items-center gap-6 md:flex">
            <Link
              href="/features"
              className="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("features")}
            </Link>
            <Link
              href="/pricing"
              className="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("pricing")}
            </Link>
          </nav>

          <div className="flex-1" />

          <LanguageSwitcher />
          <ThemeToggle />

          <AuthButton />
        </div>
      </header>

      <main className="flex-1">{children}</main>

      <footer className="border-t border-border/70">
        <div className="mx-auto w-full max-w-7xl px-4 py-8">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2.5">
              <BrandMark className="size-5 rounded-sm [&_svg]:size-3" />
              <span className="text-sm font-medium">{tCommon("appName")}</span>
            </div>
            <p className="text-sm text-muted-foreground">{tLanding("footer")}</p>
          </div>
          <div className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-2 border-t border-border/70 pt-5">
            <Link
              href="/features"
              className="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("features")}
            </Link>
            <Link
              href="/pricing"
              className="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("pricing")}
            </Link>
            <Link
              href="/security"
              className="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("security")}
            </Link>
            <Link
              href="/privacy"
              className="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("privacy")}
            </Link>
            <Link
              href="/terms"
              className="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {t("terms")}
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
