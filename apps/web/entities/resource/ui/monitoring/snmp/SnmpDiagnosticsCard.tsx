"use client";

import { useState } from "react";
import { ChevronDown, Terminal } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { Button } from "@/shared/ui/button";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import { resourcesApi } from "@/entities/resource/api/resource.api";
import { formatRelativeTime } from "@/shared/utils/formatters";

// Advanced, non-sensitive diagnostics for the SNMP collector. Hidden from the
// main flow; secrets are never part of the payload.
export function SnmpDiagnosticsCard({
  resourceId,
  monitorId,
  isFa,
}: {
  resourceId: string;
  monitorId: string;
  isFa: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const [open, setOpen] = useState(false);

  const query = useQuery({
    queryKey: ["resources", resourceId, "snmp", monitorId, "diagnostics"],
    queryFn: () => resourcesApi.snmpDiagnostics(resourceId, monitorId),
    enabled: open,
    staleTime: 15_000,
  });

  const d = query.data;

  return (
    <div className="rounded-xl border border-border/40">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between px-4 py-3 text-xs font-semibold text-muted-foreground transition-colors hover:text-foreground"
      >
        <span className="flex items-center gap-2">
          <Terminal className="size-3.5" />
          {t("Advanced Diagnostics", "تشخیص پیشرفته")}
        </span>
        <ChevronDown className={cn("size-4 transition-transform", open && "rotate-180")} />
      </button>

      {open && (
        <div className="border-t border-border/40 px-4 py-3">
          {query.isPending ? (
            <Skeleton className="h-40 w-full rounded-lg" />
          ) : d ? (
            <div className="flex flex-col gap-2">
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                <Diagnostic label={t("Collector", "کلکتور")} value={String(d.collector_version ?? "—")} ltr />
                <Diagnostic label={t("Last state", "آخرین وضعیت")} value={String(d.state ?? "—")} ltr />
                <Diagnostic label={t("SNMP version", "نسخه SNMP")} value={String(d.snmp_version ?? "—")} ltr />
                <Diagnostic label={t("Response time", "زمان پاسخ")} value={`${String(d.duration_millis ?? "—")} ms`} ltr />
              </div>
              <Diagnostic label={t("Last check", "آخرین بررسی")} value={d.last_check_at ? formatRelativeTime(String(d.last_check_at), isFa ? "fa" : "en") : "—"} />
              {d.error_code ? <Diagnostic label={t("Last error", "آخرین خطا")} value={String(d.error_code)} ltr /> : null}
              {d.partial_failures ? (
                <Diagnostic label={t("Partial failures", "شکست‌های جزئی")} value={String(Array.isArray(d.partial_failures) ? (d.partial_failures as string[]).join(", ") : d.partial_failures)} />
              ) : null}
              {d.device ? (
                <pre className="max-h-40 overflow-auto rounded-md bg-muted/40 p-3 font-mono text-[10px] text-muted-foreground" dir="ltr">
                  {JSON.stringify(d.device, null, 2)}
                </pre>
              ) : null}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">
              {t("No diagnostics available", "تشخیصی در دسترس نیست")}
            </p>
          )}
        </div>
      )}
    </div>
  );
}

function Diagnostic({ label, value, ltr }: { label: string; value: string; ltr?: boolean }) {
  return (
    <div className="flex flex-col gap-0.5 rounded-lg border border-border/40 px-3 py-1.5">
      <span className="text-[10px] text-muted-foreground">{label}</span>
      <span className="truncate text-xs font-medium tabular-nums text-foreground" dir={ltr ? "ltr" : "auto"}>
        {value}
      </span>
    </div>
  );
}
