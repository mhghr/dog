"use client";

import { useState } from "react";
import { useLocale } from "next-intl";
import { BarChart3, Bell, Settings } from "lucide-react";

import { Skeleton } from "@/shared/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useResourceMonitors, useMonitorTypes } from "@/entities/resource/hooks/use-resource";
import { ResourceHeader } from "./components/resource-header";
import { MonitorTypeList } from "./components/monitor-type-list";
import { MonitorConfig } from "./components/monitor-config";

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
        <TabsList variant="line" className="!inline-flex w-full gap-1 rounded-full bg-muted/50 p-1 sm:w-fit">
          {TABS.map((t) => (
            <TabsTrigger key={t.v} value={t.v} className="group relative flex-1 gap-2 rounded-full px-5 py-2 text-sm font-medium cursor-pointer text-muted-foreground transition-all duration-200 hover:text-foreground data-[state=active]:bg-white data-[state=active]:text-foreground data-[state=active]:shadow-sm dark:data-[state=active]:bg-background sm:flex-none after:!hidden">
              {t.i && <t.i className={`size-4 ${TAB_ICONS[t.v as keyof typeof TAB_ICONS]}`} />}
              <span>{fa ? t.lFa : t.l}</span>
            </TabsTrigger>
          ))}
        </TabsList>

        <div className="mt-6">
          <TabsContent value="monitoring">
            {mq.isPending ? <GridSkel n={4} /> : <p className="text-sm text-muted-foreground py-12 text-center">Monitoring view</p>}
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

function SettingsPanel({ resourceId, monitors }: { resourceId: string; monitors: any[] }) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const tq = useMonitorTypes();
  const [sel, setSel] = useState<string | null>(null);
  const types = tq.data?.items ?? [];
  const st = types.find((t: any) => t.id === sel) ?? null;
  const sm = st ? monitors.find((m: any) => m.monitor_type_id === st.id) : undefined;

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
