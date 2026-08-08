"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/shared/ui/button";
import { useMe } from "@/platform/auth/use-auth";
import { Link } from "@/i18n/navigation";
import { Skeleton } from "@/shared/ui/skeleton";

export function AuthButton() {
  const t = useTranslations("navigation");
  const meQuery = useMe();

  if (meQuery.isPending) {
    return <Skeleton className="h-8 w-20 rounded-lg" />;
  }

  if (meQuery.isSuccess && meQuery.data) {
    return (
      <Button asChild size="sm">
        <Link href="/login">{t("console")}</Link>
      </Button>
    );
  }

  return (
    <Button asChild size="sm">
      <Link href="/login">{t("login")}</Link>
    </Button>
  );
}
