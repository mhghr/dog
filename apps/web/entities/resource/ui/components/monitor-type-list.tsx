"use client";

import { useState } from "react";
import { useLocale } from "next-intl";
import { BookOpen, ExternalLink } from "lucide-react";
import { Switch } from "@/shared/ui/switch";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import { MONITOR_TYPE_ITEMS, type MonitorTypeItem } from "../monitor-types";
import type { Monitor } from "@/entities/resource/hooks/use-resource";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import { MonitorTypeIcon } from "./monitor-type-icon";

export function MonitorTypeList({
  types,
  monitors,
  selectedId,
  onSelect,
  isPending,
}: {
  types: MonitorTypeDef[];
  monitors: Monitor[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  isPending: boolean;
}) {
  const locale = useLocale();
  const fa = locale === "fa";
  const [toggles, setToggles] = useState<Record<string, boolean>>({});

  const fallbackById = Object.fromEntries(MONITOR_TYPE_ITEMS.map((item) => [item.id, item]));
  const fallbackByName = new Map(MONITOR_TYPE_ITEMS.map((item) => [item.title.toLowerCase(), item]));
  const fallbackByFaName = new Map(MONITOR_TYPE_ITEMS.map((item) => [item.titleFa, item]));

  const TONE_PALETTE = [
    "bg-primary/15 text-primary",
    "bg-emerald-500/15 text-emerald-500",
    "bg-amber-500/15 text-amber-500",
    "bg-blue-500/15 text-blue-500",
    "bg-violet-500/15 text-violet-500",
    "bg-cyan-500/15 text-cyan-500",
    "bg-rose-500/15 text-rose-500",
  ];

  function findFallback(t: MonitorTypeDef): MonitorTypeItem | undefined {
    return fallbackById[t.slug]
      ?? fallbackById[t.id]
      ?? fallbackByName.get(t.name?.toLowerCase())
      ?? fallbackByFaName.get(t.name);
  }

  const displayTypes: MonitorTypeItem[] = types.length > 0
    ? types.map((t, i) => {
        const fallback = findFallback(t);
        const title = fa ? (fallback?.titleFa ?? t.name) : (fallback?.title ?? t.name);
        const description = fa ? (fallback?.descriptionFa ?? t.description ?? "") : (fallback?.description ?? t.description ?? "");
        const tone = fallback?.tone ?? TONE_PALETTE[i % TONE_PALETTE.length];
        return {
          id: t.id,
          title,
          titleFa: title,
          description,
          descriptionFa: description,
          icon: () => <MonitorTypeIcon type={t.name} className="size-4" />,
          tone,
        } as unknown as MonitorTypeItem;
      })
    : (fa
        ? MONITOR_TYPE_ITEMS.map((item) => ({
            ...item,
            title: item.titleFa,
            description: item.descriptionFa,
          }))
        : MONITOR_TYPE_ITEMS);

  if (isPending) {
    return (
      <div className="panel flex h-full flex-col p-5">
        <Skeleton className="h-6 w-36" /><Skeleton className="h-4 w-48 mt-2" />
        <div className="mt-5 space-y-3">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-16 rounded-xl" />)}</div>
      </div>
    );
  }

  return (
    <div className="panel flex h-full flex-col p-5">
      <h2 className="text-lg font-semibold">{fa ? "انواع مانیتورینگ" : "Monitor Types"}</h2>
      <p className="mt-1 text-sm text-muted-foreground">
        {fa ? "فعال‌سازی و پیکربندی انواع مانیتورینگ" : "Enable and configure monitoring types for this resource."}
      </p>

      <ul className="mt-5 space-y-3">
        {displayTypes.map(({ id, title, description, icon: Icon, tone }) => {
          const m = monitors.find((x) => x.monitor_type_id === id);
          const active = m?.enabled ?? (toggles[id] ?? false);
          return (
            <li key={id}>
              <div
                role="button"
                tabIndex={0}
                onClick={() => onSelect(id)}
                onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); onSelect(id); } }}
                className={cn(
                  "flex w-full cursor-pointer items-center gap-2 rounded-xl border bg-surface/60 p-3 transition-all hover:border-primary/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  selectedId === id ? "border-primary shadow-glow" : "border-border",
                )}
              >
                <span className={cn("icon-tile border-transparent grid size-8 shrink-0 place-items-center rounded-lg", tone)}>
                  <Icon className="size-4" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">{title}</span>
                  <span className="block truncate text-xs text-muted-foreground">{description}</span>
                </span>
                <Switch
                  checked={active}
                  onCheckedChange={(v) => setToggles((s) => ({ ...s, [id]: v }))}
                  onClick={(e) => e.stopPropagation()}
                  aria-label={`Enable ${title}`}
                />
              </div>
            </li>
          );
        })}
      </ul>

      <div className="mt-4 flex items-center gap-3 rounded-xl border border-dashed border-border p-3">
        <span className="icon-tile grid size-9 shrink-0 place-items-center rounded-lg bg-secondary text-secondary-foreground">
          <BookOpen className="size-4" />
        </span>
        <span className="flex-1">
          <span className="block text-sm font-medium">{fa ? "کمک نیاز دارید؟" : "Need help?"}</span>
          <span className="block text-xs text-muted-foreground">{fa ? "آشنایی با انواع مانیتورینگ" : "Learn more about monitoring types"}</span>
        </span>
        <ExternalLink className="size-4 text-muted-foreground" />
      </div>
    </div>
  );
}
