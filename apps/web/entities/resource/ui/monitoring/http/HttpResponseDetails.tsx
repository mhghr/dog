"use client";

import { ShieldCheck, ShieldAlert, MousePointerClick, CircleDot } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import { statusLabelOf } from "./http-metrics";
import type { ProbeResult } from "@/entities/monitor/model/result";

// Response Details card: the facts of a single HTTP check — status code,
// resolved IP, content type, method, body size and the content-assertion
// outcome. Everything is read from the result's attributes/metrics, never
// parsed from strings.
export function HttpResponseDetails({
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
            {t("Response Details", "جزئیات پاسخ")}
          </CardTitle>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("Select a check to see its response details", "برای مشاهده جزئیات پاسخ، یک چک را انتخاب کنید")}
          </p>
        </CardContent>
      </Card>
    );
  }

  const attributes = result.attributes ?? {};
  const statusCode = typeof attributes.status_code === "number" ? (attributes.status_code as number) : null;
  const resolvedIp = typeof attributes.resolved_ip === "string" ? (attributes.resolved_ip as string) : null;
  const contentType = typeof attributes.content_type === "string" ? (attributes.content_type as string) : null;
  const assertionStatus = typeof attributes.assertion_status === "string" ? (attributes.assertion_status as string) : null;
  const method = typeof attributes.method === "string" ? (attributes.method as string) : null;
  const responseSizeBytes = typeof result.metrics?.response_size_bytes === "number"
    ? (result.metrics.response_size_bytes as number)
    : null;

  const statusOk = statusCode != null && statusCode >= 200 && statusCode < 300;

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="text-sm font-semibold text-foreground">
            {t("Response Details", "جزئیات پاسخ")}
          </CardTitle>
          {statusCode != null && (
            <span
              className={cn(
                "inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-semibold tabular-nums",
                statusOk ? "bg-success/12 text-success" : "bg-destructive/12 text-destructive",
              )}
              dir="ltr"
            >
              {statusLabelOf(statusCode, null)}
            </span>
          )}
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-2.5 px-4 pb-4">
        <dl className="flex flex-col gap-2 text-xs">
          <Row label={t("Resolved IP", "آدرس IP")} value={resolvedIp ?? t("—", "—")} mono />
          <Row label={t("Content type", "نوع محتوا")} value={contentType ?? t("—", "—")} />
          <Row label={t("Method", "متد")} value={method ?? t("—", "—")} mono />
          {responseSizeBytes != null && (
            <Row label={t("Body size", "اندازه بدنه")} value={formatBytes(responseSizeBytes)} />
          )}
          <Row
            label={t("Checked at", "زمان بررسی")}
            value={new Date(result.finished_at ?? result.started_at).toLocaleString(isFa ? "fa-IR" : "en-US")}
          />
        </dl>

        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <FactTile
            icon={statusOk ? <ShieldCheck className="size-3.5" /> : <ShieldAlert className="size-3.5" />}
            label={t("Status", "وضعیت")}
            value={statusOk ? t("Healthy", "سالم") : t("Failed", "ناموفق")}
            tone={statusOk ? "ok" : "bad"}
          />
          <FactTile
            icon={<MousePointerClick className="size-3.5" />}
            label={t("Assertion", "تطابق محتوا")}
            value={
              assertionStatus === "ok"
                ? t("Passed", "موفق")
                : assertionStatus === "failed"
                  ? t("Failed", "ناموفق")
                  : t("Not set", "تنظیم نشده")
            }
            tone={assertionStatus === "ok" ? "ok" : assertionStatus === "failed" ? "bad" : "muted"}
          />
        </div>

        {resolvedIp && (
          <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <CircleDot className="size-3 text-primary" />
            {t("Request reached", "درخواست به این آدرس رسید")}
            <code className="font-mono text-foreground/80" dir="ltr">{resolvedIp}</code>
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

function FactTile({
  icon,
  label,
  value,
  tone,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  tone: "ok" | "bad" | "muted";
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-2.5 rounded-lg border p-2.5",
        tone === "ok" && "border-success/20 bg-success/[0.05]",
        tone === "bad" && "border-destructive/20 bg-destructive/[0.05]",
        tone === "muted" && "border-border/60 bg-muted/30",
      )}
    >
      <span className={cn("grid size-7 shrink-0 place-items-center rounded-md", tone === "ok" && "text-success", tone === "bad" && "text-destructive", tone === "muted" && "text-muted-foreground")}>
        {icon}
      </span>
      <span className="min-w-0">
        <span className="block truncate text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">{label}</span>
        <span className="block truncate text-sm font-medium text-foreground">{value}</span>
      </span>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
