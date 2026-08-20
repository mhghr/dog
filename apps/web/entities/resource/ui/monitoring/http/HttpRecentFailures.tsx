"use client";

import { TriangleAlert } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { formatRelativeTime } from "@/shared/utils/formatters";
import type { HttpProbeHealth } from "./http-metrics";

const ERROR_TYPES = [
  { en: "DNS Failed", fa: "خطای DNS", match: "dns_resolution_failed" },
  { en: "Timeout", fa: "تایم‌اوت", match: "timeout" },
  { en: "Connection Refused", fa: "اتصال رد شد", match: "connection_refused" },
  { en: "TLS Error", fa: "خطای TLS", match: "tls" },
  { en: "Invalid Certificate", fa: "گواهی نامعتبر", match: "certificate" },
  { en: "Status Code Mismatch", fa: "عدم تطابق کد وضعیت", match: "unexpected_status_code" },
  { en: "Content Validation Failed", fa: "خطای اعتبارسنجی محتوا", match: "assertion" },
] as const;

function errorTypeOf(code: string | null): { en: string; fa: string } | null {
  if (!code) return null;
  const lower = code.toLowerCase();
  for (const type of ERROR_TYPES) {
    if (lower.includes(type.match)) return { en: type.en, fa: type.fa };
  }
  return null;
}

interface HttpRecentFailuresProps {
  probeHealth: HttpProbeHealth[];
  isFa: boolean;
}

// Recent failures across all probes, newest first. Rendered only when at
// least one probe is failing — this is the incident timeline for the monitor.
export function HttpRecentFailures({ probeHealth, isFa }: HttpRecentFailuresProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const failures = probeHealth
    .filter((p) => !p.success || p.health === "critical" || p.health === "down")
    .sort((a, b) => String(b.lastCheckedAt ?? "").localeCompare(String(a.lastCheckedAt ?? "")));

  if (failures.length === 0) return null;

  const typesSeen = new Set<string>();
  for (const f of failures) {
    const type = errorTypeOf(f.errorCode);
    if (type) typesSeen.add(isFa ? type.fa : type.en);
  }

  return (
    <Card variant="bordered" className="border-destructive/25 shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="flex items-center gap-2 text-sm font-semibold text-destructive">
          <TriangleAlert className="size-4" />
          {t("Recent Failures", "خطاهای اخیر")}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 px-4 pb-4">
        <div className="flex flex-col gap-1.5">
          {failures.map((failure) => {
            const type = errorTypeOf(failure.errorCode);
            return (
              <div
                key={failure.probeId}
                className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border border-border/50 px-3.5 py-2.5"
              >
                <span className="text-xs tabular-nums text-muted-foreground">
                  {failure.lastCheckedAt ? formatRelativeTime(failure.lastCheckedAt, isFa ? "fa" : "en") : "—"}
                </span>
                <span className="rounded-md bg-destructive/10 px-1.5 py-0.5 text-[11px] font-medium text-destructive" dir="ltr">
                  {type ? (isFa ? type.fa : type.en) : (failure.errorCode ?? "error")}
                </span>
                <span className="text-xs font-medium text-foreground" dir="auto">
                  {failure.location}
                </span>
                {failure.errorMessage && (
                  <span className="min-w-0 flex-1 truncate text-[11px] text-muted-foreground" dir="ltr">
                    {failure.errorMessage}
                  </span>
                )}
              </div>
            );
          })}
        </div>

        {typesSeen.size > 0 && (
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-[11px] font-medium text-muted-foreground">
              {t("Error types", "انواع خطا")}:
            </span>
            {ERROR_TYPES.map((errorType) => {
              const label = isFa ? errorType.fa : errorType.en;
              const active = typesSeen.has(label);
              return (
                <span
                  key={errorType.match}
                  className={
                    active
                      ? "rounded-full bg-destructive/12 px-2 py-0.5 text-[11px] font-medium text-destructive"
                      : "rounded-full bg-muted/60 px-2 py-0.5 text-[11px] text-muted-foreground"
                  }
                >
                  {label}
                </span>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
