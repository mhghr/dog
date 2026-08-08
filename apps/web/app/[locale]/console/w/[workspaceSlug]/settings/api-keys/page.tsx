import { getTranslations, setRequestLocale } from "next-intl/server";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { GearSix } from "@/shared/ui/icons";

export default async function ApikeysPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: "settings" });

  return (
    <EmptyState
      icon={GearSix}
      title={t("apiKeysTitle")}
      description={t("apiKeysBody")}
    />
  );
}
