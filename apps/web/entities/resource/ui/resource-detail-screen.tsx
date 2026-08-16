"use client";

import { useState } from "react";
import { useLocale } from "next-intl";
import { BarChart3, Bell, Settings } from "lucide-react";

import { Skeleton } from "@/shared/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { ErrorState } from "@/design-system/patterns/error-state";
import {
  isPingMonitor,
  useMonitorTypes,
  useResourceMonitors,
} from "@/entities/resource/hooks/use-resource";
import type { Monitor } from "@/entities/resource/hooks/types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import { ResourceHeader } from "./components/resource-header";
import { MonitorTypeList } from "./components/monitor-type-list";
import { MonitorConfig } from "./components/monitor-config";
import { PingMonitoringView } from "./monitoring/ping/PingMonitoringView";

const TAB_ICONS = {
  monitoring: "text-emerald-500",
  metrics: "text-blue-500",
  alerts: "text-amber-500",
  settings: "text-violet-500",
} as const;

const TABS = [
  { v: "monitoring", l: "Monitoring",   lFa: "مانیتورینگ", i: null },
  { v: "metrics",    l: "Metrics",      lFa: "متریک‌ها",   i: BarChart3 },
  { v: "alerts",     l: "Alerts",       lFa: "هشدارها",    i: Bell },
  { v: "settings",   l: "Settings",     lFa: "تنظیمات",    i: Settings },
] as const;

export function ResourceDetailScreen({ resourceId }: { resourceId: string }) {
  const locale = useLocale();
  const fa = locale === "fa";
  const mq = useResourceMonitors(resourceId);
  const monitors = mq.data?.items ?? [];

  return (
    <div className="space-y-6">
      <ResourceHeader resourceId={resourceId} />

      <Tabs defaultValue="monitoring">
        <TabsList className="!inline-flex w-full gap-1 rounded-[10px] bg-muted/70 p-1 sm:w-fit dark:bg-zinc-800/70">
          {TABS.map((t) => (
            <TabsTrigger key={t.v} value={t.v} className="group relative flex-1 gap-2 rounded-lg px-4 py-1.5 text-[13px] font-medium cursor-pointer text-muted-foreground transition-all duration-200 hover:text-foreground data-[state=active]:bg-card data-[state=active]:text-foreground data-[state=active]:shadow-[0_1px_2px_rgba(0,0,0,0.06)] dark:text-zinc-400 dark:hover:text-zinc-100 dark:data-[state=active]:bg-zinc-700 dark:data-[state=active]:text-white sm:flex-none after:!hidden">
              {t.i && <t.i className={`size-3.5 ${TAB_ICONS[t.v as keyof typeof TAB_ICONS]}`} />}
              <span>{fa ? t.lFa : t.l}</span>
            </TabsTrigger>
          ))}
        </TabsList>

        <div className="mt-6">
          <TabsContent value="monitoring">
            <MonitoringPanel resourceId={resourceId} />
          </TabsContent>
          <TabsContent value="metrics">
            <p className="text-sm text-muted-foreground py-12 text-center">{fa ? "متریک‌ها به زودی" : "Metrics coming soon"}</p>
          </TabsContent>
          <TabsContent value="alerts">
            <p className="text-sm text-muted-foreground py-12 text-center">{fa ? "هشدارها به زودی" : "Alerts coming soon"}</p>
          </TabsContent>
          <TabsContent value="settings">
            <SettingsPanel resourceId={resourceId} monitors={monitors} />
          </TabsContent>
        </div>
      </Tabs>
    </div>
  );
}

function SettingsPanel({ resourceId, monitors }: { resourceId: string; monitors: Monitor[] }) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const tq = useMonitorTypes();
  const [sel, setSel] = useState<string | null>(null);
  const types = tq.data?.items ?? [];
  const st = types.find((t: MonitorTypeDef) => t.id === sel) ?? null;
  const sm = st ? monitors.find((m: Monitor) => m.monitor_type_id === st.id) : undefined;

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-[320px_1fr]">
      <MonitorTypeList
        types={types}
        monitors={monitors}
        selectedId={sel}
        onSelect={setSel}
        isPending={tq.isPending}
      />
      <div>
        {st ? (
          <MonitorConfig resourceId={resourceId} type={st} monitor={sm} isFa={isFa} />
        ) : (
          <div className="panel flex items-center justify-center py-20">
            <p className="text-sm text-muted-foreground">{isFa ? "یک نوع مانیتور را از سمت چپ انتخاب کنید" : "Select a monitor type from the left to configure it."}</p>
          </div>
        )}
      </div>
    </div>
  );
}

function GridSkel({ n }: { n: number }) {
  return <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
    {Array.from({ length: n }).map((_, i) => <Skeleton key={i} className="h-32 rounded-2xl" />)}
  </div>;
}

function MonitoringPanel({ resourceId }: { resourceId: string }) {
  const locale = useLocale();
  const fa = locale === "fa";
  const mq = useResourceMonitors(resourceId);
  const tq = useMonitorTypes();

  const monitors = mq.data?.items ?? [];
  const types = tq.data?.items ?? [];

  const pingMonitors = monitors.filter((m) => isPingMonitor(m, types));

  if (mq.isPending || tq.isPending) return <GridSkel n={4} />;

  const failed =
    (mq.isError && !mq.isFetching) || (tq.isError && !tq.isFetching);
  if (failed) {
    return (
      <ErrorState
        onRetry={() => {
          void mq.refetch();
          void tq.refetch();
        }}
      />
    );
  }

  if (pingMonitors.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/50 py-20">
        <p className="text-sm font-medium">{fa ? "مانیتور پینگ پیکربندی نشده" : "No Ping monitor configured"}</p>
        <p className="text-sm text-muted-foreground">
          {fa
            ? "از تب تنظیمات یک مانیتور پینگ فعال و ذخیره کنید."
            : "Enable and save a Ping monitor from the Settings tab."}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {pingMonitors.map((m) => (
        <PingMonitoringView key={m.id} resourceId={resourceId} monitor={m} />
      ))}
    </div>
  );
}
