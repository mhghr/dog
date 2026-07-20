"use client";

import { use } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { MonitorForm } from "@/components/monitors/monitor-form";
import { Skeleton } from "@/components/ui/skeleton";
import { useMonitor } from "@/hooks/use-monitor";
import { useUpdateMonitor } from "@/hooks/use-monitor-mutations";
import { useRouter } from "@/i18n/navigation";
import { ApiError } from "@/lib/api-client";
import { monitorToFormValues } from "@/lib/schemas";
import type { CreateMonitorInput } from "@/types/monitor";

export default function EditMonitorPage({
  params,
}: {
  params: Promise<{ monitorId: string }>;
}) {
  const { monitorId } = use(params);

  const t = useTranslations("monitors");
  const tValidation = useTranslations("validation");
  const router = useRouter();

  const monitorQuery = useMonitor(monitorId);
  const updateMutation = useUpdateMonitor(monitorId);

  const handleSubmit = async (payload: CreateMonitorInput) => {
    try {
      await updateMutation.mutateAsync(payload);
      toast.success(tValidation("updateSuccess"));
      router.push(`/app/monitors/${monitorId}`);
    } catch (error) {
      if (!(error instanceof ApiError) || !error.fields) {
        toast.error(tValidation("genericError"));
      }
      throw error;
    }
  };

  if (monitorQuery.isPending) {
    return (
      <div className="mx-auto flex max-w-3xl flex-col gap-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-64 rounded-xl" />
        <Skeleton className="h-64 rounded-xl" />
      </div>
    );
  }

  if (monitorQuery.isError) {
    return <ErrorState onRetry={() => void monitorQuery.refetch()} />;
  }

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader title={t("editMonitor")} subtitle={monitorQuery.data.name} />
      <MonitorForm
        initialValues={monitorToFormValues(monitorQuery.data)}
        typeLocked
        submitLabel={t("form.submitSave")}
        pending={updateMutation.isPending}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
