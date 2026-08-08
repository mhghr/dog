import { useTranslations } from "next-intl";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { List } from "@/shared/ui/icons";

// ResourceLogsView is the placeholder for log streaming on a resource. The
// log pipeline is a data-stream domain; this view will consume the log SSE
// stream once the backend log ingestion is available.
export function ResourceLogsView({ resourceId }: { resourceId: string }) {
  const t = useTranslations("resources");

  return (
    <div className="flex flex-col gap-4">
      <EmptyState
        icon={List}
        title={t("logsTitle")}
        description={t("logsBody")}
      />
    </div>
  );
}
