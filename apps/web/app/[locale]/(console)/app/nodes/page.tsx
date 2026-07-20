"use client";

import { useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, MonitorCheck } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import {
  MonitorFilters,
  type MonitorFilterState,
} from "@/components/monitors/monitor-filters";
import {
  MonitorTable,
  MonitorTableSkeleton,
} from "@/components/monitors/monitor-table";
import { NodeCreateFlow } from "@/components/monitors/node-create-flow";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useMonitors } from "@/hooks/use-monitors";
import { useCreateMonitor } from "@/hooks/use-monitor-mutations";
import { ApiError } from "@/lib/api-client";
import { formatNumber } from "@/lib/formatters";
import type { CreateMonitorInput } from "@/types/monitor";

const PAGE_SIZE = 20;

export default function NodesPage() {
  const t = useTranslations("monitors");
  const tNav = useTranslations("navigation");
  const tValidation = useTranslations("validation");
  const locale = useLocale();

  const [view, setView] = useState<"list" | "add">("list");

  const [filters, setFilters] = useState<MonitorFilterState>({
    search: "",
    type: "all",
    status: "all",
  });
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);

  useEffect(() => {
    const handle = window.setTimeout(() => {
      setDebouncedSearch(filters.search.trim());
      setPage(1);
    }, 350);

    return () => window.clearTimeout(handle);
  }, [filters.search]);

  const monitorsQuery = useMonitors({
    page,
    pageSize: PAGE_SIZE,
    search: debouncedSearch,
    type: filters.type,
    status: filters.status,
  });

  const hasFilters =
    debouncedSearch !== "" || filters.type !== "all" || filters.status !== "all";

  const pagination = monitorsQuery.data?.pagination;
  const isRTL = locale === "fa";
  const PrevIcon = isRTL ? ChevronRight : ChevronLeft;
  const NextIcon = isRTL ? ChevronLeft : ChevronRight;

  const createMutation = useCreateMonitor();

  const handleSubmit = async (payload: CreateMonitorInput) => {
    try {
      await createMutation.mutateAsync(payload);
      toast.success(tValidation("createSuccess"));
      setView("list");
    } catch (error) {
      if (!(error instanceof ApiError) || !error.fields) {
        toast.error(tValidation("genericError"));
      }
      throw error;
    }
  };

  return (
    <Tabs value={view} onValueChange={(v) => setView(v as "list" | "add")}>
      <div className="mb-6 flex items-center justify-between">
        <TabsList>
          <TabsTrigger value="list">{tNav("myNodes")}</TabsTrigger>
          <TabsTrigger value="add">{tNav("addNode")}</TabsTrigger>
        </TabsList>

        {view === "list" ? (
          <Button onClick={() => setView("add")}>{t("newMonitor")}</Button>
        ) : null}
      </div>

      <TabsContent value="list">
        <MonitorFilters
          value={filters}
          onChange={(next) => {
            setFilters(next);
            if (next.type !== filters.type || next.status !== filters.status) {
              setPage(1);
            }
          }}
        />

        {monitorsQuery.isPending ? (
          <MonitorTableSkeleton />
        ) : monitorsQuery.isError ? (
          <ErrorState onRetry={() => void monitorsQuery.refetch()} />
        ) : monitorsQuery.data.items.length === 0 ? (
          hasFilters ? (
            <EmptyState title={t("noResultsTitle")} icon={MonitorCheck} />
          ) : (
            <EmptyState
              title={t("emptyTitle")}
              description={t("emptyBody")}
              icon={MonitorCheck}
              action={
                <Button onClick={() => setView("add")}>
                  {t("emptyCta")}
                </Button>
              }
            />
          )
        ) : (
          <>
            <MonitorTable monitors={monitorsQuery.data.items} />

            {pagination && pagination.total_pages > 1 ? (
              <div className="mt-4 flex items-center justify-between gap-3">
                <p className="text-sm text-muted-foreground">
                  {t("totalCount", {
                    count: formatNumber(pagination.total, locale),
                  })}
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="icon"
                    disabled={page <= 1}
                    onClick={() => setPage((current) => current - 1)}
                    aria-label="previous page"
                  >
                    <PrevIcon className="size-4" />
                  </Button>
                  <span className="text-sm tabular-nums">
                    {t("pageOf", {
                      page: formatNumber(pagination.page, locale),
                      total: formatNumber(pagination.total_pages, locale),
                    })}
                  </span>
                  <Button
                    variant="outline"
                    size="icon"
                    disabled={page >= pagination.total_pages}
                    onClick={() => setPage((current) => current + 1)}
                    aria-label="next page"
                  >
                    <NextIcon className="size-4" />
                  </Button>
                </div>
              </div>
            ) : null}
          </>
        )}
      </TabsContent>

      <TabsContent value="add">
        <div className="mx-auto max-w-3xl">
          <NodeCreateFlow
            pending={createMutation.isPending}
            onSubmit={handleSubmit}
          />
        </div>
      </TabsContent>
    </Tabs>
  );
}
