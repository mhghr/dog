"use client";

import { Activity } from "lucide-react";
import { useTranslations } from "next-intl";

import { RelativeTime } from "@/shared/ui/relative-time";
import { StatusPageActions } from "@/features/status-pages/ui/status-page-actions";
import { Skeleton } from "@/shared/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table";
import type { StatusPage } from "@/features/status-pages/model/types";

export function StatusPageTableSkeleton() {
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-border p-4">
      {Array.from({ length: 6 }).map((_, i) => (
        <Skeleton key={i} className="h-12 w-full" />
      ))}
    </div>
  );
}

export function StatusPageTable({
  statusPages,
  onEdit,
}: {
  statusPages: StatusPage[];
  onEdit: (statusPage: StatusPage) => void;
}) {
  const t = useTranslations("statusPages");
  const tCommon = useTranslations("common");

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("name")}</TableHead>
              <TableHead className="hidden sm:table-cell">{t("slug")}</TableHead>
              <TableHead>{t("enabled")}</TableHead>
              <TableHead className="hidden md:table-cell">
                <Activity className="-ms-0.5 me-1 inline size-3.5" aria-hidden />
                {t("componentsTitle")}
              </TableHead>
              <TableHead className="hidden sm:table-cell">
                <span className="sr-only">Created</span>
              </TableHead>
              <TableHead>
                <span className="sr-only">{tCommon("actions")}</span>
              </TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {statusPages.map((sp) => (
              <TableRow
                key={sp.id}
                className="hover:bg-muted/50 transition-colors"
              >
                <TableCell className="max-w-48 font-medium">
                  <div className="truncate">{sp.name}</div>
                  {sp.description ? (
                    <div className="truncate text-xs text-muted-foreground">
                      {sp.description}
                    </div>
                  ) : null}
                </TableCell>
                <TableCell
                  dir="ltr"
                  className="hidden font-mono text-xs sm:table-cell"
                >
                  {sp.slug}
                </TableCell>
                <TableCell>
                  {sp.enabled ? (
                    <span className="inline-flex items-center rounded-full bg-success/12 px-2 py-0.5 text-xs font-medium text-success">
                      {tCommon("yes")}
                    </span>
                  ) : (
                    <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                      {tCommon("no")}
                    </span>
                  )}
                </TableCell>
                <TableCell className="hidden md:table-cell tabular-nums">
                  {t("componentCount", { count: sp.component_count })}
                </TableCell>
                <TableCell className="hidden text-sm text-muted-foreground sm:table-cell">
                  <RelativeTime value={sp.created_at} />
                </TableCell>
                <TableCell className="text-end">
                  <StatusPageActions
                    statusPage={sp}
                    onEdit={() => onEdit(sp)}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
