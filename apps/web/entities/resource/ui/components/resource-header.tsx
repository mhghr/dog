"use client";

import { useEffect, useState } from "react";
import { useLocale } from "next-intl";
import { Loader2, MapPin, MoreHorizontal } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Skeleton } from "@/shared/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { cn } from "@/shared/utils/cn";
import { useResource, useUpdateResource } from "@/entities/resource/hooks/use-resource";
import type { Resource } from "@/entities/resource/model/types";
import { probeApi } from "@/entities/probe/api/probe.api";
import { apiErrorMessage } from "@/shared/api/error-message";
import { getResourceIcon } from "@/design-system/icons";

interface ResourceLocation {
  country?: string;
  city?: string;
  lat?: number;
  lon?: number;
}

// Shows the resource's location below the address. Uses the saved location
// when present, otherwise auto-detects it from the target IP via the geo-IP
// API so the location is always visible.
function ResourceLocationLine({ resource, fa }: { resource: Resource; fa: boolean }) {
  const [loc, setLoc] = useState<{ city?: string; country?: string }>(() => {
    const meta = (resource.metadata ?? {}) as { location?: { city?: string; country?: string } };
    return meta.location ?? {};
  });

  useEffect(() => {
    if (loc.city || loc.country) return;
    const ip = resource.target?.trim();
    if (!ip || !/^[\d.:]+$/.test(ip)) return;

    let cancelled = false;
    probeApi
      .geoIpLookup(ip)
      .then((res) => {
        if (!cancelled && (res.city || res.country)) {
          setLoc({ city: res.city, country: res.country });
        }
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [resource.target, loc.city, loc.country]);

  const label = [loc.city, loc.country].filter(Boolean).join(", ");

  return (
    <p className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground">
      <MapPin className="size-3 shrink-0" />
      <span dir="auto">{label || (fa ? "??????" : "Unknown")}</span>
    </p>
  );
}

function EditResourceDialog({
  resource,
  onOpenChange,
}: {
  resource: Resource;
  onOpenChange: (open: boolean) => void;
}) {
  const locale = useLocale();
  const fa = locale === "fa";
  const [name, setName] = useState(resource.name);
  const [target, setTarget] = useState(resource.target ?? "");
  const [location, setLocation] = useState<ResourceLocation>(() => {
    const meta = (resource.metadata ?? {}) as { location?: ResourceLocation };
    return meta.location ?? {};
  });
  const [detecting, setDetecting] = useState(false);
  const [pending, setPending] = useState(false);
  const update = useUpdateResource(resource.id);

  // When the target (IP/address) changes, auto-detect the location from the
  // entered IP via the geo-IP API.
  useEffect(() => {
    const ip = target.trim();
    if (!ip) return;

    const isIpLike = /^[\d.:]+$/.test(ip);
    if (!isIpLike) return;

    const handle = window.setTimeout(() => {
      setDetecting(true);
      probeApi
        .geoIpLookup(ip)
        .then((loc) => {
          if (loc.city || loc.country) {
            setLocation({ city: loc.city, country: loc.country, lat: loc.lat, lon: loc.lon });
          }
        })
        .catch(() => {
          // Private/undetected IP — leave the current location as-is.
        })
        .finally(() => setDetecting(false));
    }, 600);

    return () => window.clearTimeout(handle);
  }, [target]);

  const locationLabel = [location.city, location.country].filter(Boolean).join(", ");

  const submit = async () => {
    setPending(true);
    try {
      const metadata = { ...resource.metadata, location };
      await update.mutateAsync({ name: name.trim(), target: target.trim(), metadata });
      toast.success(fa ? "????? ??" : "Saved");
      onOpenChange(false);
    } catch (err) {
      const msg = apiErrorMessage(err, fa);
      toast.error(msg.title, { description: msg.description });
    } finally {
      setPending(false);
    }
  };

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{fa ? "?????? ??????" : "Edit resource"}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="res-name">{fa ? "???" : "Name"}</Label>
            <Input
              id="res-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              dir="auto"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="res-target">{fa ? "???? / IP" : "Address / IP"}</Label>
            <Input
              id="res-target"
              value={target}
              onChange={(e) => setTarget(e.target.value)}
              dir="ltr"
            />
            {detecting ? (
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Loader2 className="size-3 animate-spin" />
                {fa ? "?? ??? ????? ??????..." : "Detecting location..."}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                {fa ? "??????: " : "Location: "}
                <span dir="auto">{locationLabel || (fa ? "??????" : "Unknown")}</span>
              </p>
            )}
          </div>

          <div className="flex items-center justify-end gap-2 pt-1">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={pending}
              onClick={() => onOpenChange(false)}
            >
              {fa ? "??????" : "Cancel"}
            </Button>
            <Button
              size="sm"
              disabled={pending || !name.trim()}
              onClick={() => void submit()}
            >
              {pending ? (fa ? "?? ??? ?????..." : "Saving...") : (fa ? "?????" : "Save")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function ResourceHeader({ resourceId }: { resourceId: string }) {
  const locale = useLocale();
  const fa = locale === "fa";
  const { data: r, isPending } = useResource(resourceId);
  const [editOpen, setEditOpen] = useState(false);

  if (isPending) return <div className="space-y-4"><Skeleton className="h-7 w-64 rounded-lg" /><Skeleton className="h-4 w-96 rounded-lg" /></div>;
  if (!r) return null;

  const h = r.health_status ?? "unknown";
  const s = r.health_score ?? 0;
  const c = r.monitors_count ?? 0;
  const a = r.avg_response_ms;

  const hl = fa
    ? h === "healthy" ? "????" : h === "degraded" ? "?????" : h === "down" ? "?????" : "—"
    : h === "healthy" ? "Healthy" : h === "degraded" ? "Warning" : h === "down" ? "Down" : "—";

  const resIcon = getResourceIcon(r.type_category);

  const stats = [
    { label: fa ? "?????" : "Status", value: hl, tone: h === "healthy" ? "success" : h === "degraded" ? "warning" : "muted" as const },
    { label: fa ? "??????" : "Uptime", value: s > 0 ? `${s.toFixed(1)}%` : "—", tone: s >= 99 ? "success" as const : s >= 95 ? "warning" as const : "muted" as const },
    { label: fa ? "?????????" : "Monitors", value: `${c}`, tone: "info" as const },
    { label: fa ? "????" : "Response", value: a ? `${a}ms` : "—", tone: a && a < 500 ? "success" as const : a && a < 1000 ? "warning" as const : "muted" as const },
  ];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-6" dir="ltr">
        <dl className="grid shrink-0 grid-cols-2 gap-3 xl:grid-cols-4 xl:w-[36rem]">
          {stats.map((st) => (
            <div
              key={st.label}
              className="rounded-xl bg-card px-4 py-3 shadow-subtle ring-1 ring-foreground/5 transition-[border-color,box-shadow] duration-300 dark:hover:border-primary/40 dark:hover:shadow-glow dark:ring-white/10"
            >
              <dt className="text-xs text-muted-foreground">{st.label}</dt>
              <dd className={cn("mt-1 flex items-center gap-2 text-lg font-semibold",
                st.tone === "success" ? "text-success dark:neon-text-current"
                : st.tone === "warning" ? "text-warning dark:neon-text-current"
                : st.tone === "info" ? "text-info dark:neon-text-current"
                : "text-muted-foreground"
              )}>
                {st.tone === "success" && <span className="size-2 rounded-full bg-success dark:shadow-[0_0_8px_1px_var(--success)]" />}
                {st.value}
              </dd>
            </div>
          ))}
        </dl>

        <div className="flex min-w-0 items-center gap-4" dir="rtl">
          <span className={cn("grid size-14 shrink-0 place-items-center rounded-xl ring-1", resIcon.color, "ring-current/20")}>
            <resIcon.icon className="size-7" />
          </span>
          <div className="min-w-0 text-right" dir="auto">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="truncate text-xl font-bold tracking-tight">{r.name}</h1>
              <Badge className={cn("rounded-md border text-[11px] font-semibold uppercase tracking-wider px-2.5 py-0.5",
                r.status === "active" ? "border-success/25 bg-success/10 text-success" : "border-muted bg-muted/40 text-muted-foreground"
              )}>{fa ? "????" : r.status}</Badge>
              <Button
                variant="ghost"
                size="icon"
                className="size-8 text-muted-foreground hover:text-foreground"
                aria-label={fa ? "??????" : "Edit"}
                onClick={() => setEditOpen(true)}
              >
                <MoreHorizontal className="size-4" />
              </Button>
            </div>
            <p className="mt-1 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <a href={r.target?.startsWith("http") ? r.target : `https://${r.target}`}
                target="_blank" rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground text-sm" dir="ltr">
                {r.target}
              </a>
            </p>
            <ResourceLocationLine resource={r} fa={fa} />
          </div>
        </div>
      </div>

      {editOpen && <EditResourceDialog resource={r} onOpenChange={setEditOpen} />}
    </div>
  );
}
