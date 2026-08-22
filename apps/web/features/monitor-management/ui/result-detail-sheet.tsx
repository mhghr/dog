"use client";

import { useLocale, useTranslations } from "next-intl";

import { Badge } from "@/shared/ui/badge";
import { Separator } from "@/shared/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/shared/ui/sheet";
import { formatDateTime, formatDuration } from "@/shared/utils/formatters";
import { cn } from "@/shared/utils/cn";
import type { ProbeResult } from "@/entities/monitor/model/result";

// Sensitive attribute keys are never rendered in the drawer.
const HIDDEN_ATTRIBUTE_KEYS = new Set([
  "authorization",
  "cookie",
  "set-cookie",
  "token",
]);

function DetailRow({
  label,
  value,
}: {
  label: React.ReactNode;
  value: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-4 py-1.5 text-sm">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="break-all text-end font-medium">{value}</span>
    </div>
  );
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) {
    return "—";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

export function ResultDetailSheet({
  result,
  onClose,
}: {
  result: ProbeResult | null;
  onClose: () => void;
}) {
  const t = useTranslations("results");
  const locale = useLocale();

  const metrics = Object.entries(result?.metrics ?? {});
  const attributes = Object.entries(result?.attributes ?? {}).filter(
    ([key]) => !HIDDEN_ATTRIBUTE_KEYS.has(key.toLowerCase()),
  );

  return (
    <Sheet open={Boolean(result)} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{t("detailTitle")}</SheetTitle>
          {result ? (
            <SheetDescription asChild>
              <div>
                <Badge
                  className={cn(
                    "border-transparent font-medium",
                    result.success
                      ? "bg-success/12 text-success"
                      : "bg-destructive/12 text-destructive",
                  )}
                >
                  {result.success ? t("success") : t("failure")}
                </Badge>
              </div>
            </SheetDescription>
          ) : null}
        </SheetHeader>

        {result ? (
          <div className="flex flex-col gap-4 px-4 pb-8">
            <div>
              <DetailRow
                label={t("duration")}
                value={formatDuration(result.duration_millis, locale)}
              />
              <DetailRow
                label={t("startedAt")}
                value={
                  <span dir="ltr">{formatDateTime(result.started_at, locale)}</span>
                }
              />
              <DetailRow
                label={t("finishedAt")}
                value={
                  <span dir="ltr">{formatDateTime(result.finished_at, locale)}</span>
                }
              />
              {result.error_code ? (
                <DetailRow
                  label={t("errorCode")}
                  value={
                    <span dir="ltr" className="font-mono text-xs">
                      {result.error_code}
                    </span>
                  }
                />
              ) : null}
            </div>

            {result.error_message ? (
              <>
                <Separator />
                <div>
                  <p className="mb-1.5 text-sm text-muted-foreground">
                    {t("errorMessage")}
                  </p>
                  <p
                    dir="ltr"
                    className="break-all rounded-lg border border-destructive/30 bg-destructive/5 p-3 font-mono text-xs leading-relaxed"
                  >
                    {result.error_message}
                  </p>
                </div>
              </>
            ) : null}

            {metrics.length > 0 ? (
              <>
                <Separator />
                <div>
                  <p className="mb-1.5 text-sm font-medium">{t("metrics")}</p>
                  {metrics.map(([key, value]) => (
                    <DetailRow
                      key={key}
                      label={<span dir="ltr" className="font-mono text-xs">{key}</span>}
                      value={<span dir="ltr" className="font-mono text-xs">{formatValue(value)}</span>}
                    />
                  ))}
                </div>
              </>
            ) : null}

            {attributes.length > 0 ? (
              <>
                <Separator />
                <div>
                  <p className="mb-1.5 text-sm font-medium">{t("attributes")}</p>
                  {attributes.map(([key, value]) => (
                    <DetailRow
                      key={key}
                      label={<span dir="ltr" className="font-mono text-xs">{key}</span>}
                      value={<span dir="ltr" className="break-all font-mono text-xs">{formatValue(value)}</span>}
                    />
                  ))}
                </div>
              </>
            ) : null}
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
