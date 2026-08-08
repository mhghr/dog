"use client";

import { useMemo, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";
import { Plus } from "lucide-react";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { ErrorState } from "@/design-system/patterns/error-state";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Skeleton } from "@/shared/ui/skeleton";
import {
  useCreateResource,
  useResourceTypes,
  useResources,
} from "@/entities/resource/hooks/use-resource";
import type {
  Resource,
  ResourceType,
} from "@/entities/resource/model/types";
import { ResourceTypeIcon } from "@/entities/resource/ui/components/resource-type-icon";
import { Link, usePathname } from "@/i18n/navigation";
import { cn } from "@/shared/utils/cn";

function healthTone(status: string | undefined): string {
  switch (status) {
    case "healthy":
    case "up":
    case "ok":
      return "bg-success/12 text-success";
    case "degraded":
    case "warning":
      return "bg-warning/15 text-warning";
    case "down":
    case "critical":
    case "error":
      return "bg-destructive/12 text-destructive";
    default:
      return "bg-muted text-muted-foreground";
  }
}

function healthLabel(t: (key: string) => string, status: string | undefined) {
  switch (status) {
    case "healthy":
    case "up":
    case "ok":
      return t("healthy");
    case "degraded":
    case "warning":
      return t("warning");
    case "down":
    case "critical":
    case "error":
      return t("critical");
    default:
      return "—";
  }
}

function ResourceCard({ resource }: { resource: Resource }) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const t = useTranslations("resources");
  const pathname = usePathname();

  // Stay inside the active workspace console route.
  const wsMatch = pathname.match(/^\/console\/w\/([^/]+)/);
  const base = wsMatch ? `/console/w/${wsMatch[1]}` : "/app";

  return (
    <Link href={`${base}/resources/${resource.id}`} className="block h-full">
      <Card
        variant="bordered"
        className="h-full transition-all duration-200 hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5"
      >
        <CardContent className="flex h-full flex-col gap-3 p-4">
          <div className="flex items-start justify-between gap-2">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <ResourceTypeIcon
                type={resource.type_icon ?? resource.type_name ?? ""}
                className="size-6"
              />
            </div>
            <span
              className={cn(
                "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                healthTone(resource.health_status),
              )}
            >
              <span className="size-1.5 rounded-full bg-current" aria-hidden />
              {healthLabel(t, resource.health_status)}
            </span>
          </div>

          <div className="min-w-0">
            <h3 className="truncate text-sm font-semibold" dir="auto">
              {resource.name}
            </h3>
            <p className="mt-0.5 truncate text-xs text-muted-foreground" dir="ltr">
              {resource.target || resource.type_name || "—"}
            </p>
          </div>

          <div className="mt-auto flex items-center justify-between gap-2 border-t border-border/60 pt-3 text-xs text-muted-foreground">
            <span>{resource.type_name ?? "—"}</span>
            <span className="tabular-nums">
              {resource.monitors_count ?? 0}{" "}
              {isFa ? "مانیتور" : resource.monitors_count === 1 ? "monitor" : "monitors"}
            </span>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}

function ResourceGridSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {Array.from({ length: 8 }).map((_, i) => (
        <Card key={i} variant="bordered" className="h-44">
          <CardContent className="p-4">
            <Skeleton className="size-11 rounded-xl" />
            <Skeleton className="mt-3 h-4 w-2/3" />
            <Skeleton className="mt-2 h-3 w-1/2" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function AddResourceDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const t = useTranslations("resources");
  const locale = useLocale();
  const isFa = locale === "fa";
  const { data: typesData, isPending: typesPending } = useResourceTypes();
  const createResource = useCreateResource();

  const types = useMemo(() => typesData?.items ?? [], [typesData]);

  const [step, setStep] = useState<1 | 2>(1);
  const [selectedType, setSelectedType] = useState<ResourceType | null>(null);
  const [name, setName] = useState("");
  const [target, setTarget] = useState("");

  const handleClose = (v: boolean) => {
    onOpenChange(v);
    if (!v) {
      setStep(1);
      setSelectedType(null);
      setName("");
      setTarget("");
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedType) return;
    try {
      await createResource.mutateAsync({
        resource_type_id: selectedType.id,
        name: name.trim(),
        target: target.trim(),
        description: "",
        metadata: {},
      });
      toast.success(isFa ? "ریسورس اضافه شد" : "Resource added");
      handleClose(false);
    } catch {
      toast.error(isFa ? "خطا در ایجاد ریسورس" : "Failed to create resource");
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {step === 1
              ? isFa
                ? "انتخاب نوع ریسورس"
                : "Select resource type"
              : isFa
                ? "اطلاعات پایه"
                : "Basic information"}
          </DialogTitle>
        </DialogHeader>

        {step === 1 ? (
          <div>
            <p className="mb-4 text-sm text-muted-foreground">{t("subtitle")}</p>
            {typesPending ? (
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {Array.from({ length: 6 }).map((_, i) => (
                  <Skeleton key={i} className="h-28 rounded-xl" />
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {types.map((type) => (
                  <button
                    key={type.id}
                    type="button"
                    onClick={() => {
                      setSelectedType(type);
                      setStep(2);
                    }}
                    className={cn(
                      "group flex flex-col items-center justify-center gap-2 rounded-xl border border-border bg-card p-4 text-center transition-all hover:border-primary/50 hover:bg-accent",
                    )}
                  >
                    <span className="flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors group-hover:bg-primary/10 group-hover:text-primary">
                      <ResourceTypeIcon type={type.name} className="size-5" />
                    </span>
                    <span className="text-sm font-medium">{type.name}</span>
                    <span className="text-xs text-muted-foreground">{type.category}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex items-center gap-3 rounded-lg border border-border/70 bg-muted/30 p-3">
              <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <ResourceTypeIcon type={selectedType?.name ?? ""} className="size-5" />
              </span>
              <div>
                <p className="text-sm font-medium">{selectedType?.name}</p>
                <p className="text-xs text-muted-foreground">{selectedType?.category}</p>
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="resource-name">{isFa ? "نام" : "Name"}</Label>
              <Input
                id="resource-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={isFa ? "مثلاً سرور تولید" : "e.g. Production server"}
                required
                dir="auto"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="resource-target">{isFa ? "آدرس" : "Address"}</Label>
              <Input
                id="resource-target"
                value={target}
                onChange={(e) => setTarget(e.target.value)}
                placeholder={isFa ? "آدرس یا target" : "Address or target"}
                dir="ltr"
              />
            </div>

            <div className="flex items-center justify-between pt-2">
              <Button type="button" variant="ghost" onClick={() => setStep(1)}>
                {isFa ? "بازگشت" : "Back"}
              </Button>
              <Button type="submit" disabled={!name.trim() || createResource.isPending}>
                {createResource.isPending
                  ? isFa
                    ? "در حال ایجاد..."
                    : "Creating..."
                  : isFa
                    ? "ایجاد ریسورس"
                    : "Create resource"}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default function ResourcesPage() {
  const t = useTranslations("resources");
  const locale = useLocale();
  const isFa = locale === "fa";

  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [wizardOpen, setWizardOpen] = useState(false);
  const [timeoutId, setTimeoutId] = useState<ReturnType<typeof setTimeout> | null>(null);

  const handleSearchChange = (value: string) => {
    setSearch(value);
    if (timeoutId) clearTimeout(timeoutId);
    setTimeoutId(
      setTimeout(() => {
        setDebouncedSearch(value.trim());
      }, 350),
    );
  };

  const resourcesQuery = useResources({
    page: 1,
    pageSize: 60,
    search: debouncedSearch,
  });

  const resources = resourcesQuery.data?.items ?? [];

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">{t("title")}</h1>
          <p className="mt-0.5 text-sm text-muted-foreground">{t("subtitle")}</p>
        </div>
        <Button onClick={() => setWizardOpen(true)}>
          <Plus className="size-4" aria-hidden />
          {t("addResource")}
        </Button>
      </div>

      <div className="mb-5 flex items-center gap-3">
        <div className="relative w-full max-w-sm">
          <Input
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            placeholder={t("search")}
            dir="auto"
          />
        </div>
      </div>

      {resourcesQuery.isPending ? (
        <ResourceGridSkeleton />
      ) : resourcesQuery.isError ? (
        <ErrorState onRetry={() => void resourcesQuery.refetch()} />
      ) : resources.length === 0 ? (
        <EmptyState
          title={debouncedSearch ? t("noResults") : isFa ? "ریسورسی وجود ندارد" : "No resources yet"}
          description={
            debouncedSearch
              ? undefined
              : isFa
                ? "اولین ریسورس خود را اضافه کنید"
                : "Add your first resource to start monitoring"
          }
          action={
            !debouncedSearch ? (
              <Button onClick={() => setWizardOpen(true)}>
                <Plus className="size-4" aria-hidden />
                {t("addResource")}
              </Button>
            ) : undefined
          }
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {resources.map((resource) => (
            <ResourceCard key={resource.id} resource={resource} />
          ))}
        </div>
      )}

      <AddResourceDialog open={wizardOpen} onOpenChange={setWizardOpen} />
    </div>
  );
}
