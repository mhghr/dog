"use client";

import { useMemo, useState, useEffect, useCallback } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Resolver } from "react-hook-form";
import { useForm } from "react-hook-form";
import { Loader2, MapPin } from "lucide-react";
import { useTranslations } from "next-intl";

import {
  NumberField,
  SwitchField,
  TextField,
} from "@/components/monitors/form-fields";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { MonitorThresholdFields } from "@/components/monitors/monitor-threshold-fields";
import { ApiError } from "@/lib/api-client";
import { getMonitorDefinition, getMonitorFormField, MONITOR_TYPES } from "@/features/monitors/core/registry";
import {
  buildMonitorPayload,
  buildProbeConfig,
  createMonitorFormSchema,
  defaultFormValues,
  type MonitorFormValues,
} from "@/lib/schemas";
import { cn } from "@/lib/utils";
import type { CreateMonitorInput, MonitorType } from "@/types/monitor";

const FORM_STEP_FIELDS: Array<Array<keyof MonitorFormValues>> = [
  ["name", "target"],
  ["interval_seconds", "timeout_millis", "retries", "enabled"],
  [],
];

// ---------------------------------------------------------------------------
// Step 1 – Type selection
// ---------------------------------------------------------------------------

function TypeSelection({
  selected,
  onSelect,
}: {
  selected: MonitorType | null;
  onSelect: (type: MonitorType) => void;
}) {
  const tTypes = useTranslations("types");
  const tLanding = useTranslations("landing");

  return (
    <div className="grid grid-cols-1 gap-3 rounded-xl border border-border/70 bg-card/40 p-4 sm:grid-cols-2">
      {MONITOR_TYPES.map((type) => {
        const Icon = getMonitorDefinition(type).icon;
        const isSelected = selected === type;

        return (
          <button
            key={type}
            type="button"
            onClick={() => onSelect(type)}
            aria-pressed={isSelected}
            className={cn(
              "group flex items-start gap-4 rounded-lg border p-4 text-start transition-[color,background-color,border-color,box-shadow,transform] active:scale-[0.98]",
              isSelected
                ? "border-primary bg-primary/5 shadow-sm ring-2 ring-primary/20"
                : "border-border/70 bg-white hover:border-primary/40 hover:bg-primary/5 dark:bg-card",
            )}
          >
            <span
              className={cn(
                "flex size-10 shrink-0 items-center justify-center rounded-lg transition-colors",
                isSelected
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground",
              )}
            >
              <Icon className="size-5" aria-hidden />
            </span>
            <div className="min-w-0">
              <span className="text-sm font-semibold">{tTypes(type)}</span>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                {tLanding(`typeDesc.${type}`)}
              </p>
            </div>
          </button>
        );
      })}
    </div>
  );
}

function GeoLocation({ target }: { target: string }) {
  const [location, setLocation] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const resolveHost = useCallback((host: string): string => {
    const cleaned = host.replace(/^https?:\/\//, "").replace(/:\d+$/, "").replace(/\/.*$/, "");
    return cleaned;
  }, []);

  useEffect(() => {
    const host = resolveHost(target);
    if (!host || host.length < 4) {
      setLocation(null);
      return;
    }

    const timer = setTimeout(async () => {
      setLoading(true);
      try {
        const resp = await fetch(`http://ip-api.com/json/${encodeURIComponent(host)}?fields=city,country`);
        if (!resp.ok) throw new Error("geo lookup failed");
        const data = await resp.json();
        if (data.city && data.country) {
          setLocation(`${data.city}, ${data.country}`);
        } else {
          setLocation(null);
        }
      } catch {
        setLocation(null);
      } finally {
        setLoading(false);
      }
    }, 600);

    return () => clearTimeout(timer);
  }, [target, resolveHost]);

  if (!target || target.length < 4) return null;

  return (
    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <MapPin className="size-3 shrink-0" />
      {loading ? (
        <span className="inline-flex items-center gap-1">
          <span className="size-2.5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          Detecting location...
        </span>
      ) : location ? (
        <span className="text-primary">{location}</span>
      ) : null}
    </div>
  );
}

interface CreateFormProps {
  type: MonitorType;
  step: number;
  pending: boolean;
  onSubmit: (payload: CreateMonitorInput) => Promise<void>;
  onNext: () => void;
  onBack: () => void;
}

function CreateForm({
  type,
  step,
  pending,
  onSubmit,
  onNext,
  onBack,
}: CreateFormProps) {
  const t = useTranslations("monitors");
  const tCommon = useTranslations("common");
  const tFields = useTranslations("monitors.fields");
  const tValidation = useTranslations("validation");

  const schema = useMemo(
    () => createMonitorFormSchema((key, values) => tValidation(key, values)),
    [tValidation],
  );
  const monitorDefinition = getMonitorDefinition(type);

  const form = useForm<MonitorFormValues>({
    resolver: zodResolver(schema) as unknown as Resolver<MonitorFormValues>,
    defaultValues: defaultFormValues(type),
    mode: "onBlur",
  });

  const watchedValues = form.watch();

  const configPreview = useMemo(
    () => JSON.stringify(buildProbeConfig(watchedValues), null, 2),
    [watchedValues],
  );

  const handleNext = async () => {
    const fields = FORM_STEP_FIELDS[step];
    if (fields.length === 0) {
      return;
    }
    const valid = await form.trigger(fields);
    if (valid) {
      onNext();
    }
  };

  const handleFormSubmit = form.handleSubmit(async (values) => {
    try {
      await onSubmit(buildMonitorPayload(values));
    } catch (error) {
      if (error instanceof ApiError && error.fields) {
        for (const [field, messages] of Object.entries(error.fields)) {
          const formField = getMonitorFormField(values.type, field);
          if (formField && messages.length > 0) {
            form.setError(formField, { type: "server", message: messages[0] });
          }
        }
      }
      throw error;
    }
  });

  const isLastStep = step === 2;

  return (
    <div className="min-h-[280px]">
      <form
        onSubmit={(event) => {
          void handleFormSubmit(event).catch(() => undefined);
        }}
        noValidate
        className="flex flex-col gap-6"
      >
        {step === 0 && (
          <Card>
            <CardHeader>
              <CardTitle>{t("form.general")}</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2">
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
                placeholder={t(`form.targetPlaceholder.${type}`)}
              />
              <div className="flex flex-col gap-1.5">
                <Label className="text-xs">{t("form.location")}</Label>
                <GeoLocation target={watchedValues.target || ""} />
              </div>
            </CardContent>
          </Card>
        )}

        {step === 1 && (
          <Card>
            <CardHeader>
              <CardTitle>{t("form.scheduling")}</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <NumberField
                form={form}
                name="interval_seconds"
                label={tFields("intervalSeconds")}
                min={10}
              />
              <NumberField
                form={form}
                name="timeout_millis"
                label={tFields("timeoutMillis")}
                min={100}
                max={60000}
              />
              <NumberField
                form={form}
                name="retries"
                label={tFields("retries")}
                min={0}
                max={5}
              />
              <div className="sm:col-span-3">
                <SwitchField form={form} name="enabled" label={tFields("enabled")} />
              </div>
            </CardContent>
          </Card>
        )}

        {step === 2 && (
          <Card>
            <CardHeader>
              <CardTitle>{t("form.probeConfig")}</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-6">
              <monitorDefinition.ConfigFields form={form} />
              <MonitorThresholdFields type={type} form={form} />

              <div className="flex flex-col gap-1.5">
                <Label>{t("form.configSummary")}</Label>
                <pre
                  dir="ltr"
                  className="max-h-56 overflow-auto rounded-lg border border-border bg-secondary/50 p-3 font-mono text-xs leading-relaxed"
                >
                  {configPreview}
                </pre>
              </div>
            </CardContent>
          </Card>
        )}

        <div className="flex items-center justify-between">
          <Button
            type="button"
            variant="outline"
            onClick={onBack}
            disabled={pending}
          >
            {tCommon("back")}
          </Button>

          {isLastStep ? (
            <Button type="submit" disabled={pending} className="min-w-36">
              {pending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {t("form.submitCreate")}
            </Button>
          ) : (
            <Button type="button" onClick={handleNext}>
              {tCommon("next")}
            </Button>
          )}
        </div>
      </form>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main exported component
// ---------------------------------------------------------------------------

export function NodeCreateFlow({
  pending,
  onSubmit,
}: {
  pending: boolean;
  onSubmit: (payload: CreateMonitorInput) => Promise<void>;
}) {
  const t = useTranslations("monitors");
  const tCommon = useTranslations("common");
  const [step, setStep] = useState(0);
  const [selectedType, setSelectedType] = useState<MonitorType | null>(null);

  return (
    <div className="flex flex-col gap-6">
      {step === 0 ? (
        <div className="flex flex-col gap-4">
          <p className="text-sm font-medium">{t("form.selectTypePrompt")}</p>
          <TypeSelection selected={selectedType} onSelect={setSelectedType} />
          <div className="flex justify-end">
            <Button onClick={() => setStep(1)} disabled={!selectedType}>
              {tCommon("next")}
            </Button>
          </div>
        </div>
      ) : (
        <CreateForm
          key={selectedType}
          type={selectedType!}
          step={step - 1}
          pending={pending}
          onSubmit={onSubmit}
          onNext={() => setStep((current) => current + 1)}
          onBack={() => setStep((current) => current - 1)}
        />
      )}
    </div>
  );
}
