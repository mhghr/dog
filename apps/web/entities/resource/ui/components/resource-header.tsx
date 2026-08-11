"use client";

import { useLocale } from "next-intl";
import { MoreHorizontal } from "lucide-react";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import { useResource } from "@/entities/resource/hooks/use-resource";
import { getResourceIcon } from "@/design-system/icons";

export function ResourceHeader({ resourceId }: { resourceId: string }) {
  const locale = useLocale();
  const fa = locale === "fa";
  const { data: r, isPending } = useResource(resourceId);

  if (isPending) return <div className="space-y-4"><Skeleton className="h-7 w-64 rounded-lg" /><Skeleton className="h-4 w-96 rounded-lg" /></div>;
  if (!r) return null;

  const h = r.health_status ?? "unknown";
  const s = r.health_score ?? 0;
  const c = r.monitors_count ?? 0;
  const a = r.avg_response_ms;

  const hl = fa
    ? h === "healthy" ? "سالم" : h === "degraded" ? "هشدار" : h === "down" ? "پایین" : "—"
    : h === "healthy" ? "Healthy" : h === "degraded" ? "Warning" : h === "down" ? "Down" : "—";

  const resIcon = getResourceIcon(r.type_category);

  const stats = [
    { label: fa ? "وضعیت" : "Status", value: hl, tone: h === "healthy" ? "success" : h === "degraded" ? "warning" : "muted" as const },
    { label: fa ? "دسترسی" : "Uptime", value: s > 0 ? `${s.toFixed(1)}%` : "—", tone: s >= 99 ? "success" as const : s >= 95 ? "warning" as const : "muted" as const },
    { label: fa ? "مانیتورها" : "Monitors", value: `${c}`, tone: "info" as const },
    { label: fa ? "پاسخ" : "Response", value: a ? `${a}ms` : "—", tone: a && a < 500 ? "success" as const : a && a < 1000 ? "warning" as const : "muted" as const },
  ];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-end gap-3">
        <Button variant="outline" size="icon" className="size-8" aria-label={fa ? "بیشتر" : "More"}>
          <MoreHorizontal className="size-4" />
        </Button>
      </div>

      <div className="flex flex-wrap items-start justify-between gap-6" dir="ltr">
        <dl className="grid shrink-0 grid-cols-2 gap-3 xl:grid-cols-4 xl:w-[36rem]">
          {stats.map((st) => (
            <div key={st.label} className="panel px-4 py-3">
              <dt className="text-xs text-muted-foreground">{st.label}</dt>
              <dd className={cn("mt-1 flex items-center gap-2 text-lg font-semibold",
                st.tone === "success" ? "text-emerald-500"
                : st.tone === "warning" ? "text-amber-500"
                : st.tone === "info" ? "text-blue-500"
                : "text-muted-foreground"
              )}>
                {st.tone === "success" && <span className="size-2 rounded-full bg-emerald-500" />}
                {st.value}
              </dd>
            </div>
          ))}
        </dl>

        <div className="flex min-w-0 items-center gap-4" dir="rtl">
          <span className={cn("grid size-14 shrink-0 place-items-center rounded-xl ring-1", resIcon.color, "ring-current/20")}>
            <resIcon.icon className="size-7" />
          </span>
          <div className="min-w-0 text-right" dir="auto">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="truncate text-xl font-bold tracking-tight">{r.name}</h1>
              <Badge className={cn("rounded-md border text-[11px] font-semibold uppercase tracking-wider px-2.5 py-0.5",
                r.status === "active" ? "border-emerald-500/25 bg-emerald-500/10 text-emerald-500" : "border-muted bg-muted/40 text-muted-foreground"
              )}>{fa ? "فعال" : r.status}</Badge>
            </div>
            <p className="mt-1 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <a href={r.target?.startsWith("http") ? r.target : `https://${r.target}`}
                target="_blank" rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground text-sm" dir="ltr">
                {r.target}
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
