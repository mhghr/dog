"use client";

import { useRef, useState } from "react";
import { useLocale } from "next-intl";
import { Activity, BarChart3, Bell, Plus, Settings } from "lucide-react";

import { Skeleton } from "@/shared/ui/skeleton";
import { Button } from "@/shared/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";
import { cn } from "@/shared/utils/cn";
import { ErrorState } from "@/design-system/patterns/error-state";
import {
  isDnsMonitor,
  isHttpMonitor,
  isPingMonitor,
  isSnmpMonitor,
  isTcpMonitor,
  isTlsMonitor,
  useMonitorTypes,
  useResource,
  useResourceMonitors,
} from "@/entities/resource/hooks/use-resource";
import type { Monitor } from "@/entities/resource/hooks/types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import { ResourceHeader } from "./components/resource-header";
import { MonitorTypeList } from "./components/monitor-type-list";
import { MonitoringSettingsForm } from "./settings";
import { ResourceSummary, useMonitorSummaryCards } from "./components/monitor-summary";
import { PingMonitoringView } from "./monitoring/ping/PingMonitoringView";
import { HttpMonitoringView } from "./monitoring/http/HttpMonitoringView";
import { TcpMonitoringView } from "./monitoring/tcp/TcpMonitoringView";
import { DnsMonitoringView } from "./monitoring/dns/DnsMonitoringView";
import { TlsMonitoringView } from "./monitoring/tls/TlsMonitoringView";
import { SnmpMonitoringView } from "./monitoring/snmp/SnmpMonitoringView";
import { SnmpWizard } from "./monitoring/snmp/wizard/SnmpWizard";

const TAB_ICONS = {
  monitoring: "text-primary",
  metrics: "text-blue-500",
  alerts: "text-amber-500",
  settings: "text-violet-500",
} as const;

const TABS = [
  { v: "monitoring", l: "Monitoring", lFa: "مانیتورینگ", i: Activity },
  { v: "metrics", l: "Metrics", lFa: "متریک‌ها", i: BarChart3 },
  { v: "alerts", l: "Alerts", lFa: "هشدارها", i: Bell },
  { v: "settings", l: "Settings", lFa: "تنظیمات", i: Settings },
] as const;

type TabValue = (typeof TABS)[number]["v"];

export function ResourceDetailScreen({ resourceId }: { resourceId: string }) {
  const locale = useLocale();
  const fa = locale === "fa";
  const mq = useResourceMonitors(resourceId);
  const monitors = mq.data?.items ?? [];
  const [activeTab, setActiveTab] = useState<TabValue>("monitoring");

  return (
    <div className="space-y-6">
      <ResourceHeader resourceId={resourceId} />

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as TabValue)}
        orientation="vertical"
        className="items-start gap-4"
      >
        <TabsList className="fixed top-40 z-20 start-[var(--sidebar-width)] flex w-14 flex-col overflow-hidden rounded-none border border-border/60 bg-card/80 shadow-lg backdrop-blur dark:bg-card/60">
          {TABS.map((t, index) => (
            <Tooltip key={t.v}>
              <TooltipTrigger asChild>
                <TabsTrigger
                  value={t.v}
                  className={cn(
                    "group relative grid h-11 w-full flex-none place-items-center rounded-none text-muted-foreground transition-[color,transform] duration-300 hover:scale-105 hover:bg-accent/70 hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 data-[state=active]:text-primary",
                    index < TABS.length - 1 && "border-b border-border/60",
                  )}
                >
                  <span
                    aria-hidden
                    className="absolute inset-0 rounded-none bg-primary/10 opacity-0 shadow-[0_0_20px_-6px_var(--primary)] transition-opacity duration-300 group-data-[state=active]:opacity-100"
                  />
                  <span
                    aria-hidden
                    className="absolute inset-y-2 start-0.5 w-0.5 rounded-none bg-primary opacity-0 shadow-[0_0_8px_1px_var(--primary)] transition-opacity duration-300 group-data-[state=active]:opacity-100"
                  />
                  <t.i className={`relative size-5 transition-transform duration-300 group-hover:scale-110 ${TAB_ICONS[t.v as keyof typeof TAB_ICONS]}`} />
                </TabsTrigger>
              </TooltipTrigger>
              <TooltipContent side={fa ? "left" : "right"}>
                <span>{fa ? t.lFa : t.l}</span>
              </TooltipContent>
            </Tooltip>
          ))}
        </TabsList>

        <div className="min-w-0 flex-1 ps-16">
          <TabsContent value="monitoring">
            <MonitoringPanel resourceId={resourceId} onManage={() => setActiveTab("settings")} />
          </TabsContent>
          <TabsContent value="metrics">
            <p className="text-sm text-muted-foreground py-12 text-center">{fa ? "متریک‌ها به‌زودی" : "Metrics coming soon"}</p>
          </TabsContent>
          <TabsContent value="alerts">
            <p className="text-sm text-muted-foreground py-12 text-center">{fa ? "هشدارها به‌زودی" : "Alerts coming soon"}</p>
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
  const resourceQuery = useResource(resourceId);
  const [sel, setSel] = useState<string | null>(null);
  const types = tq.data?.items ?? [];
  const st = types.find((t: MonitorTypeDef) => t.id === sel) ?? null;
  const sm = st ? monitors.find((m: Monitor) => m.monitor_type_id === st.id) : undefined;
  const target = sm?.resource_target ?? resourceQuery.data?.target ?? "";

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
          st.executor_key === "snmp" && !sm ? (
            <SnmpWizard
              resourceId={resourceId}
              resource={resourceQuery.data}
              type={st}
              isFa={isFa}
              onDone={() => {
                void tq.refetch();
              }}
            />
          ) : (
            <MonitoringSettingsForm
              key={`${st.id}:${sm?.id ?? "new"}`}
              resourceId={resourceId}
              type={st}
              monitor={sm}
              target={target}
              isFa={isFa}
            />
          )
        ) : (
          <div className="panel flex items-center justify-center py-20">
            <p className="text-sm text-muted-foreground">
              {isFa
                ? "برای پیکربندی، از فهرست سمت راست یک نوع مانیتور را انتخاب کنید."
                : "Select a monitor type from the left to configure it."}
            </p>
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

function MonitoringPanel({ resourceId, onManage }: { resourceId: string; onManage: () => void }) {
  const locale = useLocale();
  const fa = locale === "fa";
  const mq = useResourceMonitors(resourceId);
  const tq = useMonitorTypes();

  const monitors = mq.data?.items ?? [];
  const types = tq.data?.items ?? [];
  const { cards } = useMonitorSummaryCards(resourceId, monitors, types);

  const [activeId, setActiveId] = useState<string | null>(null);
  const [highlightId, setHighlightId] = useState<string | null>(null);
  const highlightTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const scrollToMonitor = (monitorId: string) => {
    setActiveId(monitorId);
    document.getElementById(`monitor-${monitorId}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
    setHighlightId(monitorId);
    if (highlightTimer.current) clearTimeout(highlightTimer.current);
    highlightTimer.current = setTimeout(() => setHighlightId(null), 1800);
  };

  const pingMonitors = monitors.filter((m) => isPingMonitor(m, types));
  const httpMonitors = monitors.filter((m) => isHttpMonitor(m, types));
  const tcpMonitors = monitors.filter((m) => isTcpMonitor(m, types));
  const dnsMonitors = monitors.filter((m) => isDnsMonitor(m, types));
  const tlsMonitors = monitors.filter((m) => isTlsMonitor(m, types));
  const snmpMonitors = monitors.filter((m) => isSnmpMonitor(m, types));

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

  const hasAny =
    pingMonitors.length > 0 ||
    httpMonitors.length > 0 ||
    tcpMonitors.length > 0 ||
    dnsMonitors.length > 0 ||
    tlsMonitors.length > 0 ||
    snmpMonitors.length > 0;

  if (!hasAny) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/50 py-20">
        <p className="text-sm font-medium">{fa ? "هیچ مانیتوری پیکربندی نشده است" : "No monitor configured"}</p>
        <p className="text-sm text-muted-foreground">
          {fa
            ? "برای شروع، از تب تنظیمات یک مانیتور پینگ، HTTP، TCP، DNS، SSL یا SNMP را فعال و ذخیره کنید."
            : "Enable and save a Ping, HTTP, TCP, DNS, SSL or SNMP monitor from the Settings tab."}
        </p>
        <Button type="button" size="sm" className="mt-2" onClick={onManage}>
          <Plus className="size-4" />
          {fa ? "افزودن مانیتورینگ" : "Add Monitoring"}
        </Button>
      </div>
    );
  }

  const sectionProps = (monitorId: string) => ({
    id: `monitor-${monitorId}`,
    className: cn(
      "scroll-mt-28 rounded-2xl transition-[box-shadow,border-color] duration-500",
      highlightId === monitorId && "border border-primary/50 shadow-[0_0_0_3px_var(--primary)/15]",
    ),
  });

  return (
    <div className="flex flex-col gap-6">
      {/* Sticky summary: one card per active monitoring type. */}
      <ResourceSummary
        cards={cards}
        isFa={fa}
        activeId={activeId}
        onSelect={scrollToMonitor}
      />

      {pingMonitors.map((m) => (
        <section key={m.id} {...sectionProps(m.id)}>
          <PingMonitoringView resourceId={resourceId} monitor={m} />
        </section>
      ))}
      {httpMonitors.map((m) => (
        <section key={m.id} {...sectionProps(m.id)}>
          <HttpMonitoringView resourceId={resourceId} monitor={m} />
        </section>
      ))}
      {tcpMonitors.map((m) => (
        <section key={m.id} {...sectionProps(m.id)}>
          <TcpMonitoringView resourceId={resourceId} monitor={m} />
        </section>
      ))}
      {dnsMonitors.map((m) => (
        <section key={m.id} {...sectionProps(m.id)}>
          <DnsMonitoringView resourceId={resourceId} monitor={m} />
        </section>
      ))}
      {tlsMonitors.map((m) => (
        <section key={m.id} {...sectionProps(m.id)}>
          <TlsMonitoringView resourceId={resourceId} monitor={m} />
        </section>
      ))}
      {snmpMonitors.map((m) => (
        <section key={m.id} {...sectionProps(m.id)}>
          <SnmpMonitoringView resourceId={resourceId} monitor={m} />
        </section>
      ))}
    </div>
  );
}
