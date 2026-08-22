"use client";

import { TriangleAlert, Info } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { StatusBadge } from "@/design-system/components/status-badge";
import { cn } from "@/shared/utils/cn";
import {
  FAILURE_STAGE_INFO,
  failureStageLabel,
  failureToneOf,
  statusLabelOf,
} from "./http-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

// Failure Analysis card: shows the failing check's error code, the
// machine-readable failure stage (attributes.failure_stage) and a
// human-readable explanation + remediation hint. Never parses error strings.
export function HttpFailureAnalysis({
  result,
  isFa,
}: {
  result: ProbeResult | null;
  isFa: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  if (!result) {
    return (
      <Card variant="bordered" className="h-full shadow-subtle">
        <CardHeader className="px-5 pt-4">
          <CardTitle className="text-sm font-semibold text-foreground">
            {t("Failure Analysis", "تحلیل خطا")}
          </CardTitle>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("Select a failed check to see its details", "برای مشاهده جزئیات، یک چک ناموفق را انتخاب کنید")}
          </p>
        </CardContent>
      </Card>
    );
  }

  const stage = typeof result.attributes?.failure_stage === "string"
    ? (result.attributes.failure_stage as string)
    : null;
  const errorCode = result.error_code ?? null;
  const statusCode = typeof result.attributes?.status_code === "number"
    ? (result.attributes.status_code as number)
    : null;
  const tone = failureToneOf(errorCode);
  const info = stage ? FAILURE_STAGE_INFO[stage] : undefined;

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="flex-row items-center justify-between gap-3 px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Failure Analysis", "تحلیل خطا")}
        </CardTitle>
        <StatusBadge
          tone={tone}
          label={failureStageLabel(stage, isFa)}
        />
      </CardHeader>
      <CardContent className="flex flex-col gap-3 px-4 pb-4">
        {!result.success ? (
          <div
            className={cn(
              "rounded-lg border p-3",
              tone === "destructive" ? "border-destructive/20 bg-destructive/[0.05]" : "border-warning/25 bg-warning/[0.06]",
            )}
          >
            <div className="flex items-start gap-2.5">
              <TriangleAlert
                className={cn("mt-0.5 size-4 shrink-0", tone === "destructive" ? "text-destructive" : "text-warning")}
              />
              <div className="min-w-0">
                <p className={cn("text-sm font-semibold", tone === "destructive" ? "text-destructive" : "text-warning")}>
                  {info ? (isFa ? info.title.fa : info.title.en) : errorCode ?? statusLabelOf(statusCode, null)}
                </p>
                {info && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {isFa ? info.detail.fa : info.detail.en}
                  </p>
                )}
              </div>
            </div>
          </div>
        ) : (
          <div className="flex items-start gap-2.5 rounded-lg border border-success/20 bg-success/[0.05] p-3">
            <Info className="mt-0.5 size-4 shrink-0 text-success" />
            <p className="text-sm text-foreground">
              {t("This check succeeded", "این چک موفق بود")}
            </p>
          </div>
        )}

        <dl className="flex flex-col gap-2 text-xs">
          <Row
            label={t("Error code", "کد خطا")}
            value={errorCode ?? t("—", "—")}
            mono
          />
          <Row
            label={t("Stage", "مرحله")}
            value={failureStageLabel(stage, isFa)}
          />
          {statusCode != null && (
            <Row
              label={t("HTTP status", "وضعیت HTTP")}
              value={statusLabelOf(statusCode, null)}
              mono
            />
          )}
        </dl>

        {info && (
          <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
              {t("Suggested", "پیشنهاد")}
            </p>
            <p className="mt-1 text-xs text-foreground/80">{isFa ? info.hint.fa : info.hint.en}</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="shrink-0 text-muted-foreground">{label}</dt>
      <dd
        className={cn("min-w-0 truncate text-end font-medium text-foreground", mono && "font-mono")}
        dir={mono ? "ltr" : "auto"}
      >
        {value}
      </dd>
    </div>
  );
}
