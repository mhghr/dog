"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useMonitors } from "@/hooks/use-monitors";
import {
  useCreateStatusPage,
  useStatusPage,
  useUpdateStatusPage,
} from "@/hooks/use-status-pages";
import { ApiError } from "@/lib/api-client";
import type { Monitor } from "@/types/monitor";
import type { StatusPageInput } from "@/types/status-page";

const SLUG_REGEX = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

interface ComponentEntry {
  monitor_id: string;
  display_name: string;
}

interface InitialFormData {
  name: string;
  slug: string;
  description: string;
  enabled: boolean;
  components: ComponentEntry[];
}

function buildFormErrors(
  name: string,
  slug: string,
  components: ComponentEntry[],
  slugHint: string,
  componentsHint: string,
  tNameMin: string,
  tNameMax: string,
): Record<string, string> {
  const errs: Record<string, string> = {};

  if (!name || name.length < 2) errs.name = tNameMin;
  else if (name.length > 200) errs.name = tNameMax;

  if (!slug) errs.slug = slugHint;
  else if (slug.length < 2 || slug.length > 100) errs.slug = slugHint;
  else if (!SLUG_REGEX.test(slug)) errs.slug = slugHint;

  if (components.length === 0) errs.components = componentsHint;

  return errs;
}

function StatusPageForm({
  initial,
  monitors,
  onSubmit,
}: {
  initial: InitialFormData;
  monitors: Monitor[];
  onSubmit: (input: StatusPageInput) => Promise<void>;
}) {
  const t = useTranslations("statusPages");
  const tCommon = useTranslations("common");
  const tValidation = useTranslations("validation");

  const [name, setName] = useState(initial.name);
  const [slug, setSlug] = useState(initial.slug);
  const [description, setDescription] = useState(initial.description);
  const [enabled, setEnabled] = useState(initial.enabled);
  const [components, setComponents] = useState<ComponentEntry[]>(
    initial.components,
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<string | null>(null);

  const slugHint = t("slugHint");
  const componentsHint = t("componentsHint");

  const selectedIds = useMemo(
    () => new Set(components.map((c) => c.monitor_id)),
    [components],
  );

  function toggleMonitor(monitorId: string) {
    setComponents((prev) => {
      const idx = prev.findIndex((c) => c.monitor_id === monitorId);
      if (idx !== -1) {
        return prev.filter((c) => c.monitor_id !== monitorId);
      }
      return [...prev, { monitor_id: monitorId, display_name: "" }];
    });
  }

  function setDisplayName(monitorId: string, displayName: string) {
    setComponents((prev) =>
      prev.map((c) =>
        c.monitor_id === monitorId ? { ...c, display_name: displayName } : c,
      ),
    );
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setServerError(null);

    const fieldErrors = buildFormErrors(
      name,
      slug,
      components,
      slugHint,
      componentsHint,
      tValidation("nameMin"),
      tValidation("nameMax"),
    );
    if (Object.keys(fieldErrors).length > 0) {
      setErrors(fieldErrors);
      return;
    }
    setErrors({});

    const input: StatusPageInput = {
      name: name.trim(),
      slug: slug.trim(),
      description: description.trim(),
      enabled,
      components: components.map((c) => ({
        monitor_id: c.monitor_id,
        display_name: c.display_name.trim(),
      })),
    };

    try {
      await onSubmit(input);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 409 && err.code === "duplicate") {
          setErrors((prev) => ({ ...prev, slug: t("slugTaken") }));
          return;
        }
        if (err.status === 422 && err.fields) {
          const mapped: Record<string, string> = {};
          for (const [field, messages] of Object.entries(err.fields)) {
            if (messages.length > 0) {
              mapped[field] = messages[0];
            }
          }
          setErrors(mapped);
          return;
        }
      }
      setServerError(tCommon("errorTitle"));
    }
  }

  return (
    <form
      id="sp-form"
      onSubmit={(e) => {
        void handleSubmit(e);
      }}
      className="flex flex-col gap-4"
    >
      <div className="grid gap-1.5">
        <Label htmlFor="sp-name">{t("name")}</Label>
        <Input
          id="sp-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          maxLength={200}
          aria-invalid={errors.name ? true : undefined}
        />
        {errors.name ? (
          <p className="text-xs text-destructive">{errors.name}</p>
        ) : null}
      </div>

      <div className="grid gap-1.5">
        <Label htmlFor="sp-slug">{t("slug")}</Label>
        <Input
          id="sp-slug"
          dir="ltr"
          className="font-mono"
          value={slug}
          onChange={(e) => setSlug(e.target.value.toLowerCase())}
          maxLength={100}
          aria-invalid={errors.slug ? true : undefined}
        />
        {errors.slug ? (
          <p className="text-xs text-destructive">{errors.slug}</p>
        ) : (
          <p className="text-xs text-muted-foreground">{slugHint}</p>
        )}
      </div>

      <div className="grid gap-1.5">
        <Label htmlFor="sp-description">{t("description")}</Label>
        <Textarea
          id="sp-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
        />
      </div>

      <div className="flex items-center justify-between">
        <Label htmlFor="sp-enabled">{t("enabled")}</Label>
        <Switch
          id="sp-enabled"
          checked={enabled}
          onCheckedChange={setEnabled}
        />
      </div>

      <div className="grid gap-2">
        <Label>{t("componentsTitle")}</Label>
        <p className="text-xs text-muted-foreground">{componentsHint}</p>

        {monitors.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("noMonitors")}
          </p>
        ) : (
          <div className="max-h-48 space-y-1 overflow-y-auto rounded-lg border border-border p-2">
            {monitors.map((monitor) => {
              const isSelected = selectedIds.has(monitor.id);
              const entry = components.find(
                (c) => c.monitor_id === monitor.id,
              );

              return (
                <div key={monitor.id}>
                  <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1.5 hover:bg-muted/50">
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => toggleMonitor(monitor.id)}
                      className="mt-0.5"
                    />
                    <span className="text-sm leading-5">
                      {monitor.name}
                    </span>
                  </label>

                  {isSelected ? (
                    <div className="ms-8 pb-1.5 pe-2">
                      <Input
                        dir="auto"
                        className="h-7 text-xs"
                        placeholder={t("displayNamePlaceholder")}
                        value={entry?.display_name ?? ""}
                        onChange={(e) =>
                          setDisplayName(monitor.id, e.target.value)
                        }
                      />
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}

        {errors.components ? (
          <p className="text-xs text-destructive">{errors.components}</p>
        ) : null}
      </div>

      {serverError ? (
        <p className="text-sm text-destructive">{serverError}</p>
      ) : null}
    </form>
  );
}

function StatusPageFormFooter({
  onCancel,
  isPending,
  isEdit,
}: {
  onCancel: () => void;
  isPending: boolean;
  isEdit: boolean;
}) {
  const tCommon = useTranslations("common");

  return (
    <DialogFooter>
      <Button variant="outline" onClick={onCancel} disabled={isPending}>
        {tCommon("cancel")}
      </Button>
      <Button type="submit" form="sp-form" disabled={isPending} className="min-w-20">
        {isPending ? "..." : isEdit ? tCommon("save") : tCommon("create")}
      </Button>
    </DialogFooter>
  );
}

export function StatusPageDialog({
  open,
  onOpenChange,
  statusPageId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  statusPageId?: string;
}) {
  const t = useTranslations("statusPages");

  const isEdit = Boolean(statusPageId);

  const fullPageQuery = useStatusPage(statusPageId ?? "");
  const fullPage = fullPageQuery.data;

  const monitorsQuery = useMonitors({ pageSize: 100 });
  const monitors = monitorsQuery.data?.items ?? [];

  const createMutation = useCreateStatusPage();
  const updateMutation = useUpdateStatusPage(statusPageId ?? "");

  const mutation = isEdit ? updateMutation : createMutation;

  const isPageLoading = isEdit && fullPageQuery.isPending;

  function getInitialData(): InitialFormData {
    if (isEdit && fullPage) {
      return {
        name: fullPage.name,
        slug: fullPage.slug,
        description: fullPage.description,
        enabled: fullPage.enabled,
        components: (fullPage.components ?? []).map((c) => ({
          monitor_id: c.monitor_id,
          display_name: c.display_name,
        })),
      };
    }
    return {
      name: "",
      slug: "",
      description: "",
      enabled: true,
      components: [],
    };
  }

  // Key changes when data source changes; form remounts with fresh initial state
  const formKey = isEdit
    ? `${statusPageId}-${fullPage ? "ready" : "loading"}`
    : "create";

  const initialData = getInitialData();

  function handleDialogOpenChange(next: boolean) {
    if (!next && mutation.isPending) return;
    onOpenChange(next);
  }

  async function handleFormSubmit(input: StatusPageInput) {
    if (isEdit) {
      await updateMutation.mutateAsync(input);
      toast.success(t("updateSuccess"));
    } else {
      await createMutation.mutateAsync(input);
      toast.success(t("createSuccess"));
    }
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogContent
        className="sm:max-w-lg"
        showCloseButton={!mutation.isPending}
      >
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t("editPage") : t("newPage")}
          </DialogTitle>
          <DialogDescription className="sr-only">
            {isEdit ? t("editPage") : t("newPage")}
          </DialogDescription>
        </DialogHeader>

        {isPageLoading ? (
          <div className="flex flex-col gap-4 py-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        ) : (
          <StatusPageForm
            key={formKey}
            initial={initialData}
            monitors={monitors}
            onSubmit={handleFormSubmit}
          />
        )}

        {!isPageLoading ? (
          <StatusPageFormFooter
            onCancel={() => handleDialogOpenChange(false)}
            isPending={mutation.isPending}
            isEdit={isEdit}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
