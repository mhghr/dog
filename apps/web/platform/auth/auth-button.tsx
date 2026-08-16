"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/shared/ui/button";
import { useAuth } from "@/platform/auth/auth-provider";
import { Link } from "@/i18n/navigation";

export function AuthButton() {
  const t = useTranslations("navigation");
  const { isLoaded, isSignedIn } = useAuth();

  // The root layout resolves the session server-side, so isLoaded is normally
  // true in the very first render. The skeleton only appears when the server
  // genuinely could not determine the auth state (no provider, static
  // fallback) — it never hides a real Login/Console state.
  if (!isLoaded) {
    return (
      <Button asChild size="sm" variant="outline" aria-hidden>
        <span>{t("login")}</span>
      </Button>
    );
  }

  if (isSignedIn) {
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
