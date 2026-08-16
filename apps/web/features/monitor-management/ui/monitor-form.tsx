"use client";

import { useMemo } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Resolver } from "react-hook-form";
import { useForm } from "react-hook-form";
import { useTranslations } from "next-intl";
import { FileText, Clock, SlidersHorizontal, CircleCheck, AlertTriangle, CircleAlert } from "lucide-react";

import {
  NumberField,
  SwitchField,
  TextField,
} from "@/features/monitor-management/ui/form-fields";
import { Button } from "@/shared/ui/button";
import { getMonitorDefinition, MONITOR_TYPE_GROUPS } from "@/plugins/monitoring/core/registry";
import {
  buildMonitorPayload,
  createMonitorFormSchema,
  defaultFormValues,
  type MonitorFormValues,
} from "@/features/monitor-management/schemas/schemas";
import { mapServerErrors } from "@/features/monitor-management/schemas/submit-helpers";
import { cn } from "@/shared/utils/cn";
import type { CreateMonitorInput, MonitorType } from "@/entities/monitor/model/types";

interface MonitorFormProps {
  initialValues?: MonitorFormValues;
  typeLocked?: boolean;
  submitLabel: string;
  pending: boolean;
  onSubmit: (payload: CreateMonitorInput) => Promise<void>;
}

export function MonitorForm({
  initialValues,
  typeLocked = false,
  submitLabel,
  pending,
  onSubmit,
}: MonitorFormProps) {
  const t = useTranslations("monitors");
  const tTypes = useTranslations("types");
  const tGroups = useTranslations("typeGroups");
  const tFields = useTranslations("monitors.fields");
  const tValidation = useTranslations("validation");

  const schema = useMemo(
    () => createMonitorFormSchema((key, values) => tValidation(key, values)),
    [tValidation],
  );

  const form = useForm<MonitorFormValues>({
    resolver: zodResolver(schema) as unknown as Resolver<MonitorFormValues>,
    defaultValues: initialValues ?? defaultFormValues("http"),
    mode: "onBlur",
  });

  const monitorType = form.watch("type");
  const monitorDefinition = getMonitorDefinition(monitorType);

  const selectType = (nextType: MonitorType) => {
    if (typeLocked) return;
    form.setValue("type", nextType);
    form.setValue("interval_seconds", getMonitorDefinition(nextType).defaultIntervalSeconds);
    form.clearErrors();
  };

  const handleSubmit = form.handleSubmit(async (values) => {
    try {
      await onSubmit(buildMonitorPayload(values));
    } catch (error) {
      mapServerErrors(form, values.type, error);
      throw error;
    }
  });

  return (
    <form
      onSubmit={(event) => {
        void handleSubmit(event).catch(() => undefined);
      }}
      noValidate
      className="flex flex-col gap-5 max-w-4xl mx-auto"
    >
      {/* ── Type Selector ── */}
      <section>
        <h2 className="mb-3 text-sm font-semibold tracking-tight">{t("form.typeTitle")}</h2>
        <div className="flex flex-col gap-4">
          {MONITOR_TYPE_GROUPS.map((group) => (
            <div key={group.key}>
              <p className="mb-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground/60">
                {tGroups(group.key)}
              </p>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                {group.types.map((availableType) => {
                  const definition = getMonitorDefinition(availableType);
                  const Icon = definition.icon;
                  const selected = monitorType === availableType;

                  return (
                    <button
                      key={availableType}
                      type="button"
                      disabled={typeLocked && !selected}
                      onClick={() => selectType(availableType)}
                      aria-pressed={selected}
                      className={cn(
                        "relative flex items-center gap-2.5 rounded-xl border px-3.5 py-3 text-start transition-all duration-150",
                        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1",
                        selected
                          ? "border-primary/60 bg-primary/[0.06] shadow-sm shadow-primary/5"
                          : "border-border/70 bg-card/50 text-muted-foreground hover:border-primary/30 hover:bg-card hover:text-foreground",
                        typeLocked && !selected && "pointer-events-none opacity-30",
                      )}
                    >
                      <span
                        className={cn(
                          "flex size-8 shrink-0 items-center justify-center rounded-lg transition-colors",
                          selected
                            ? "bg-primary/15 text-primary"
                            : "bg-muted/60 text-muted-foreground",
                        )}
                      >
                        <Icon className="size-4" aria-hidden />
                      </span>
                      <span className="truncate text-sm font-medium">
                        {tTypes(availableType)}
                      </span>
                      {selected ? (
                        <span className="absolute end-2 top-2 size-1.5 rounded-full bg-primary" />
                      ) : null}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* ── General ── */}
      <section className="rounded-xl border border-border/60 bg-card/40 p-5 sm:p-6">
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold tracking-tight">
          <FileText className="size-4 text-primary" aria-hidden />
          {t("form.general")}
        </h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <TextField
            form={form}
            name="name"
            label={t("name")}
            placeholder={t("form.namePlaceholder")}
          />
          <TextField
            form={form}
            name="target"
            label={t("target")}
            dir="ltr"
            placeholder={t(`form.targetPlaceholder.${monitorType}`)}
          />
        </div>
      </section>

      {/* ── Scheduling ── */}
      <section className="rounded-xl border border-border/60 bg-card/40 p-5 sm:p-6">
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold tracking-tight">
          <Clock className="size-4 text-primary" aria-hidden />
          {t("form.scheduling")}
        </h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <NumberField
            form={form}
            name="interval_seconds"
            label={tFields("intervalSeconds")}
            min={10}
            suffix="s"
          />
          <NumberField
            form={form}
            name="timeout_millis"
            label={tFields("timeoutMillis")}
            min={100}
            max={60000}
            suffix="ms"
          />
          <NumberField
            form={form}
            name="retries"
            label={tFields("retries")}
            min={0}
            max={5}
            suffix="×"
          />
        </div>
        <div className="mt-4">
          <SwitchField form={form} name="enabled" label={tFields("enabled")} />
        </div>
      </section>

      {/* ── Probe Config ── */}
      <section className="rounded-xl border border-border/60 bg-card/40 p-5 sm:p-6">
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold tracking-tight">
          <SlidersHorizontal className="size-4 text-primary" aria-hidden />
          {monitorType === "ping" ? t("form.pingTitle") : t("form.probeConfig")}
        </h2>
        <monitorDefinition.ConfigFields form={form} />
      </section>

      {/* ── Health Legend ── */}
      {monitorType === "ping" ? (
        <section className="flex flex-wrap items-center gap-4 rounded-xl border border-border/60 bg-card/40 px-5 py-3.5 text-xs">
          <span className="font-medium text-foreground/80">{t("form.healthLevels")}</span>
          <span className="flex items-center gap-1.5 text-emerald-600"><CircleCheck className="size-3.5" />{t("form.healthy")}</span>
          <span className="flex items-center gap-1.5 text-amber-600"><AlertTriangle className="size-3.5" />{t("form.degraded")}</span>
          <span className="flex items-center gap-1.5 text-red-600"><CircleAlert className="size-3.5" />{t("form.down")}</span>
        </section>
      ) : null}

      {/* ── Submit ── */}
      <div className="flex items-center justify-between gap-4 rounded-xl border border-border/60 bg-card/40 px-6 py-5">
        <div className="flex flex-col gap-1">
          <p className="text-sm font-medium">{t("form.reviewBeforeSave")}</p>
          <p className="text-[11px] text-muted-foreground">{t("form.reviewHint")}</p>
        </div>
        <Button type="submit" disabled={pending} size="lg" className="min-w-44 shadow-sm">
          {pending ? (
            <span className="flex items-center gap-2">
              <span className="size-3.5 animate-spin rounded-full border-2 border-primary-foreground/30 border-t-primary-foreground" />
              {t("saving")}
            </span>
          ) : (
            submitLabel
          )}
        </Button>
      </div>
    </form>
  );
}
