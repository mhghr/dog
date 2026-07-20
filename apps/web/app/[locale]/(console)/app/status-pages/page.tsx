"use client";

import { useState } from "react";
import { Activity } from "lucide-react";
import { useTranslations } from "next-intl";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import {
  StatusPageTable,
  StatusPageTableSkeleton,
} from "@/components/status-pages/status-page-table";
import { StatusPageDialog } from "@/components/status-pages/status-page-dialog";
import { Button } from "@/components/ui/button";
import { useStatusPages } from "@/hooks/use-status-pages";

export default function StatusPagesPage() {
  const t = useTranslations("statusPages");

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const statusPagesQuery = useStatusPages();

  function handleCreate() {
    setEditingId(null);
    setDialogOpen(true);
  }

  function handleEdit(id: string) {
    setEditingId(id);
    setDialogOpen(true);
  }

  const statusPages = statusPagesQuery.data?.items ?? [];

  return (
    <div>
      <PageHeader
        title={t("title")}
        subtitle={t("subtitle")}
        actions={
          <Button onClick={handleCreate}>{t("newPage")}</Button>
        }
      />

      {statusPagesQuery.isPending ? (
        <StatusPageTableSkeleton />
      ) : statusPagesQuery.isError ? (
        <ErrorState onRetry={() => void statusPagesQuery.refetch()} />
      ) : statusPages.length === 0 ? (
        <EmptyState
          title={t("empty")}
          description={t("emptyBody")}
          icon={Activity}
          action={
            <Button onClick={handleCreate}>
              {t("newPage")}
            </Button>
          }
        />
      ) : (
        <StatusPageTable
          statusPages={statusPages}
          onEdit={(sp) => handleEdit(sp.id)}
        />
      )}

      <StatusPageDialog
        open={dialogOpen}
        onOpenChange={(next) => {
          setDialogOpen(next);
          if (!next) setEditingId(null);
        }}
        statusPageId={editingId ?? undefined}
      />
    </div>
  );
}
