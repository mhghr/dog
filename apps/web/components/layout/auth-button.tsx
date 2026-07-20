"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { useMe } from "@/hooks/use-auth";
import { Link } from "@/i18n/navigation";
import { Skeleton } from "@/components/ui/skeleton";

export function AuthButton() {
  const t = useTranslations("navigation");
  const meQuery = useMe();

  if (meQuery.isPending) {
    return <Skeleton className="h-8 w-20 rounded-lg" />;
  }

  if (meQuery.isSuccess && meQuery.data) {
    return (
      <Button asChild size="sm">
        <Link href="/app/dashboard">{t("console")}</Link>
      </Button>
    );
  }

  return (
    <Button asChild variant="ghost" size="sm">
      <Link href="/login">{t("login")}</Link>
    </Button>
  );
}
