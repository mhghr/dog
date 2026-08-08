import { getTranslations, setRequestLocale } from "next-intl/server";

import { BrandMark } from "@/shared/ui/brand-mark";

export default async function AuthLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);

  const tCommon = await getTranslations({ locale, namespace: "common" });

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-grid-faint [mask-image:radial-gradient(ellipse_60%_50%_at_50%_35%,black,transparent)]"
      />
      <div className="relative flex w-full max-w-sm flex-col gap-6">
        <div className="flex items-center justify-center gap-2.5">
          <BrandMark />
          <span className="text-sm font-semibold tracking-tight">
            {tCommon("appName")}
          </span>
        </div>
        {children}
      </div>
    </div>
  );
}
