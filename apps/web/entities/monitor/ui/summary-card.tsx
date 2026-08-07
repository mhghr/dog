import type { ReactNode } from "react";

import { Card, CardContent } from "@/components/ui/card";

export function MonitorSummaryCard({
  label,
  value,
  detail,
}: {
  label: string;
  value: string;
  detail?: ReactNode;
}) {
  return (
    <Card className="border-border/60 bg-card/55 shadow-none transition-colors hover:border-primary/20 hover:bg-card/80">
      <CardContent className="flex min-h-20 flex-col justify-between px-3 py-2.5">
        <span className="truncate text-[11px] text-muted-foreground">{label}</span>
        <div className="mt-1.5">
          <span className="text-lg font-semibold leading-none tabular-nums" dir="ltr">{value}</span>
          {detail ? <div className="mt-1 truncate text-[11px] text-muted-foreground">{detail}</div> : null}
        </div>
      </CardContent>
    </Card>
  );
}
