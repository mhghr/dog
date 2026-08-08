import { getTranslations, setRequestLocale } from "next-intl/server";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { ShieldCheck } from "@/shared/ui/icons";

export default async function RolesPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: "settings" });

  return (
    <EmptyState
      icon={ShieldCheck}
      title={t("rolesTitle")}
      description={t("rolesBody")}
    />
  );
}
