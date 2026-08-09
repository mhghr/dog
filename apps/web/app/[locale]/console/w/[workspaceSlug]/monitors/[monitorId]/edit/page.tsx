"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useMutation, useQuery } from "@tanstack/react-query";

import { MonitorForm } from "@/features/monitor-management/ui/monitor-form";
import { monitorToFormValues } from "@/features/monitor-management/schemas/schemas";
import { monitorApi } from "@/entities/monitor/api/monitor.api";
import type { CreateMonitorInput } from "@/entities/monitor/model/types";
import { Skeleton } from "@/shared/ui/skeleton";
import { useConsoleBase } from "@/widgets/console-shell/use-console-base";
import { ApiError } from "@/shared/api";

interface PageProps {
  params: Promise<{ monitorId: string }>;
}

export default function MonitorEditPage({ params }: PageProps) {
  const t = useTranslations("monitors");
  const router = useRouter();
  const base = useConsoleBase();

  return <MonitorEditContent monitorId={(params as unknown as { monitorId: string }).monitorId} base={base} router={router} t={t} />;
}

function MonitorEditContent({
  monitorId,
  base,
  router,
  t,
}: {
  monitorId: string;
  base: string;
  router: ReturnType<typeof useRouter>;
  t: ReturnType<typeof useTranslations<"monitors">>;
}) {
  const { data: monitor, isPending } = useQuery({
    queryKey: ["monitors", monitorId],
    queryFn: () => monitorApi.getById(monitorId),
    enabled: Boolean(monitorId),
  });

  const editMutation = useMutation({
    mutationFn: (payload: CreateMonitorInput) => monitorApi.update(monitorId, payload),
    onSuccess: () => router.push(`${base}/monitors/${monitorId}`),
    onError: (error) => {
      if (error instanceof ApiError) return;
    },
  });

  if (isPending) return <Skeleton className="h-96 rounded-xl" />;
  if (!monitor) return <p className="py-16 text-center text-sm text-muted-foreground">{t("notFound")}</p>;

  return (
    <div className="mx-auto max-w-2xl py-6">
      <h1 className="text-xl font-semibold tracking-tight mb-6">{t("editMonitor")}</h1>
      <MonitorForm
        initialValues={monitorToFormValues(monitor)}
        typeLocked
        submitLabel={t("save")}
        pending={editMutation.isPending}
        onSubmit={async (payload) => {
          await editMutation.mutateAsync(payload);
        }}
      />
    </div>
  );
}
