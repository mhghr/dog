"use client";

import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { NodeCreateFlow } from "@/components/monitors/node-create-flow";
import { useCreateMonitor } from "@/hooks/use-monitor-mutations";
import { useRouter } from "@/i18n/navigation";
import { ApiError } from "@/lib/api-client";
import type { CreateMonitorInput } from "@/types/monitor";

export default function NewMonitorPage() {
  const tValidation = useTranslations("validation");
  const router = useRouter();

  const createMutation = useCreateMonitor();

  const handleSubmit = async (payload: CreateMonitorInput) => {
    try {
      await createMutation.mutateAsync(payload);
      toast.success(tValidation("createSuccess"));
      router.push("/app/nodes");
    } catch (error) {
      if (!(error instanceof ApiError) || !error.fields) {
        toast.error(tValidation("genericError"));
      }
      throw error;
    }
  };

  return (
    <div className="mx-auto max-w-3xl">
      <NodeCreateFlow
        pending={createMutation.isPending}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
