import { getTranslations, setRequestLocale } from "next-intl/server";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { EnvelopeSimple } from "@/shared/ui/icons";

export default async function BillingPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: "settings" });

  return (
    <EmptyState
      icon={EnvelopeSimple}
      title={t("billingTitle")}
      description={t("billingBody")}
    />
  );
}
