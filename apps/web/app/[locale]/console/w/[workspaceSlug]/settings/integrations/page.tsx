import { getTranslations, setRequestLocale } from "next-intl/server";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { PlugsConnected } from "@/shared/ui/icons";

export default async function IntegrationsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: "settings" });

  return (
    <EmptyState
      icon={PlugsConnected}
      title={t("integrationsTitle")}
      description={t("integrationsBody")}
    />
  );
}
