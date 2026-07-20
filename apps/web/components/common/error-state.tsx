"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Warning } from "@/lib/icons";

interface ErrorStateProps {
  title?: string;
  onRetry?: () => void;
}

export function ErrorState({ title, onRetry }: ErrorStateProps) {
  const t = useTranslations("common");

  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-destructive/30 bg-destructive/5 px-6 py-16 text-center">
      <Warning className="size-8 text-destructive" aria-hidden />
      <p className="font-medium">{title ?? t("errorTitle")}</p>
      {onRetry ? (
        <Button variant="outline" size="sm" onClick={onRetry}>
          {t("retry")}
        </Button>
      ) : null}
    </div>
  );
}
