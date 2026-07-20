"use client";

import { useLocale, useTranslations } from "next-intl";

import { RelativeTime } from "@/components/common/relative-time";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDuration } from "@/lib/formatters";
import { cn } from "@/lib/utils";
import type { ProbeResult } from "@/types/result";

export function ResultTableSkeleton() {
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-border p-4">
      {Array.from({ length: 5 }).map((_, index) => (
        <Skeleton key={index} className="h-10 w-full" />
      ))}
    </div>
  );
}

export function ResultTable({
  results,
  onSelect,
}: {
  results: ProbeResult[];
  onSelect: (result: ProbeResult) => void;
}) {
  const t = useTranslations("results");
  const locale = useLocale();

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("title")}</TableHead>
              <TableHead>{t("startedAt")}</TableHead>
              <TableHead>{t("duration")}</TableHead>
              <TableHead className="hidden sm:table-cell">{t("errorCode")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {results.map((result) => (
              <TableRow
                key={result.id}
                onClick={() => onSelect(result)}
                tabIndex={0}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    onSelect(result);
                  }
                }}
                className="cursor-pointer hover:bg-muted/50 transition-colors"
              >
                <TableCell>
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
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  <RelativeTime value={result.started_at} />
                </TableCell>
                <TableCell className="text-sm tabular-nums" dir="ltr">
                  {formatDuration(result.duration_millis, locale)}
                </TableCell>
                <TableCell
                  dir="ltr"
                  className="hidden max-w-48 truncate font-mono text-xs text-muted-foreground sm:table-cell"
                >
                  {result.error_code ?? "—"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
