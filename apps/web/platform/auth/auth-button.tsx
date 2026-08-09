"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/shared/ui/button";
import { useMe } from "@/platform/auth/use-auth";
import { Link } from "@/i18n/navigation";

export function AuthButton({ hasToken }: { hasToken?: boolean }) {
  const t = useTranslations("navigation");
  const meQuery = useMe({ enabled: hasToken });

  // hasToken = cookie exists; meQuery confirms validity.
  // Fall through to login when cookie is missing or token is invalid.
  const isAuthenticated = hasToken && meQuery.isSuccess && meQuery.data;

  if (isAuthenticated) {
    return (
      <Button asChild size="sm">
        <Link href="/app/dashboard">{t("console")}</Link>
      </Button>
    );
  }

  return (
    <Button asChild size="sm">
      <Link href="/login">{t("login")}</Link>
    </Button>
  );
}
