import { getTranslations, setRequestLocale } from "next-intl/server";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { UserCircle } from "@/shared/ui/icons";

export default async function MembersPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: "settings" });

  return (
    <EmptyState
      icon={UserCircle}
      title={t("membersTitle")}
      description={t("membersBody")}
    />
  );
}
