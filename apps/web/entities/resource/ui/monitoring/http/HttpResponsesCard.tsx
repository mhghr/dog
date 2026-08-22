"use client";

import { TriangleAlert } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";

export interface HttpResponseBucket {
  code: number;
  count: number;
}

interface CategoryRow {
  key: string;
  label: string;
  fa: string;
  gradient: string;
  count: number;
}

function categoryOf(code: number): string {
  if (code >= 500) return "5xx";
  if (code >= 400) return "4xx";
  if (code >= 300) return "3xx";
  return "2xx";
}

// HTTP Responses distribution over the selected range. Status codes are
// grouped into three rows (2xx / 4xx / 5xx) so success and server errors are
// compared at a glance, with the specific error codes (500, 503, ...) listed
// separately on top.
export function HttpResponsesCard({
  buckets,
  rangeLabel,
  isFa,
}: {
  buckets: HttpResponseBucket[];
  rangeLabel: string;
  isFa: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const total = buckets.reduce((sum, b) => sum + b.count, 0);

  const errors = buckets
    .filter((b) => b.code >= 400)
    .sort((a, b) => b.count - a.count);
  const errorTotal = errors.reduce((sum, b) => sum + b.count, 0);

  const categories: CategoryRow[] = [
    { key: "2xx", label: "2xx", fa: "2xx", gradient: "bg-gradient-to-r from-emerald-500/80 to-emerald-400", count: 0 },
    { key: "4xx", label: "4xx", fa: "4xx", gradient: "bg-gradient-to-r from-amber-500/80 to-amber-400", count: 0 },
    { key: "5xx", label: "5xx", fa: "5xx", gradient: "bg-gradient-to-r from-rose-500/80 to-rose-400", count: 0 },
  ];
  for (const bucket of buckets) {
    const key = categoryOf(bucket.code);
    const row = categories.find((c) => c.key === key);
    if (row) row.count += bucket.count;
  }

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("HTTP Responses", "پاسخ‌های HTTP")} ({rangeLabel})
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2.5 px-4 pb-4">
        {total === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {t("No status codes recorded in this range", "در این بازه کد وضعیتی ثبت نشده است")}
          </p>
        ) : (
          <>
            {errors.length > 0 && (
              <div className="mb-1 flex flex-col gap-2 rounded-lg border border-destructive/20 bg-destructive/[0.04] p-3">
                <div className="flex items-center justify-between gap-2">
                  <span className="flex items-center gap-1.5 text-[11px] font-semibold text-destructive">
                    <TriangleAlert className="size-3.5" />
                    {t("Errors", "خطاها")}
                  </span>
                  <span className="text-[11px] tabular-nums text-destructive" dir="ltr">
                    {errorTotal} ({Math.round((errorTotal / total) * 100)}%)
                  </span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {errors.map((error) => (
                    <span
                      key={error.code}
                      className="inline-flex items-center gap-1 rounded-md bg-destructive/10 px-2 py-0.5 text-[11px] font-medium tabular-nums text-destructive"
                    >
                      <span dir="ltr">{error.code}</span>
                      <span dir="ltr">×{error.count}</span>
                    </span>
                  ))}
                </div>
              </div>
            )}

            {categories
              .filter((c) => c.key === "5xx" || c.count > 0)
              .map((row) => {
                const pct = (row.count / total) * 100;
                const width = row.count > 0 ? Math.max(pct, 1.5) : 0;
                return (
                  <div key={row.key} className="flex items-center gap-2.5">
                    <span className="w-10 shrink-0 text-xs font-semibold tabular-nums text-foreground" dir="ltr">
                      {row.key}
                    </span>
                    <div className="h-3 min-w-0 flex-1 overflow-hidden rounded-full bg-muted/40">
                      <div
                        className={`h-full rounded-full transition-[width] duration-500 ease-out ${row.gradient}`}
                        style={{ width: `${width}%` }}
                      />
                    </div>
                    <span className="w-24 shrink-0 text-right text-xs tabular-nums text-muted-foreground" dir="ltr">
                      {Math.round(pct)}%
                    </span>
                    <span className="w-10 shrink-0 text-right text-[11px] tabular-nums text-muted-foreground/70" dir="ltr">
                      {row.count}
                    </span>
                  </div>
                );
              })}
          </>
        )}
      </CardContent>
    </Card>
  );
}
