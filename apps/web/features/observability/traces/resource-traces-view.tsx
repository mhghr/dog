import { useTranslations } from "next-intl";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { TreeStructure } from "@/shared/ui/icons";

// ResourceTracesView is the placeholder for distributed tracing on a
// resource. Traces are a data-stream domain consumed from the realtime layer.
export function ResourceTracesView({ resourceId }: { resourceId: string }) {
  const t = useTranslations("resources");

  return (
    <div className="flex flex-col gap-4">
      <EmptyState
        icon={TreeStructure}
        title={t("tracesTitle")}
        description={t("tracesBody")}
      />
    </div>
  );
}
