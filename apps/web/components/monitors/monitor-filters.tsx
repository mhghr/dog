"use client";

import { Search } from "lucide-react";
import { useTranslations } from "next-intl";

import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MONITOR_TYPES } from "@/lib/monitor-meta";
import type { MonitorStatus, MonitorType } from "@/types/monitor";

export interface MonitorFilterState {
  search: string;
  type: MonitorType | "all";
  status: MonitorStatus | "all";
}

const STATUSES: MonitorStatus[] = ["up", "down", "unknown", "paused"];

export function MonitorFilters({
  value,
  onChange,
}: {
  value: MonitorFilterState;
  onChange: (next: MonitorFilterState) => void;
}) {
  const t = useTranslations("monitors");
  const tTypes = useTranslations("types");
  const tStatus = useTranslations("status");

  return (
    <div className="mb-6 flex flex-col items-center gap-3 sm:flex-row sm:justify-center">
      <div className="relative w-full sm:max-w-xs">
        <Search
          className="pointer-events-none absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <Input
          value={value.search}
          onChange={(event) => onChange({ ...value, search: event.target.value })}
          className="ps-9"
          aria-label={t("searchPlaceholder")}
        />
      </div>

      <div className="flex w-full gap-2 sm:w-auto">
        <Select
          value={value.type}
          onValueChange={(type) =>
            onChange({ ...value, type: type as MonitorFilterState["type"] })
          }
        >
          <SelectTrigger className="flex-1 sm:w-40" aria-label={t("type")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("allTypes")}</SelectItem>
            {MONITOR_TYPES.map((monitorType) => (
              <SelectItem key={monitorType} value={monitorType}>
                {tTypes(monitorType)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={value.status}
          onValueChange={(status) =>
            onChange({ ...value, status: status as MonitorFilterState["status"] })
          }
        >
          <SelectTrigger className="flex-1 sm:w-36" aria-label={t("status")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("allStatuses")}</SelectItem>
            {STATUSES.map((status) => (
              <SelectItem key={status} value={status}>
                {tStatus(status)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
