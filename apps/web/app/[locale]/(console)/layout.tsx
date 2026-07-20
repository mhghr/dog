import { setRequestLocale } from "next-intl/server";

import { ConsoleShell } from "@/components/layout/console-shell";

export default async function ConsoleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);

  return <ConsoleShell>{children}</ConsoleShell>;
}
