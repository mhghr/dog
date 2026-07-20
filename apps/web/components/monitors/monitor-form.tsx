"use client";

import { useMemo } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Resolver } from "react-hook-form";
import { useForm } from "react-hook-form";
import { useTranslations } from "next-intl";

import {
  NumberField,
  SwitchField,
  TextField,
} from "@/components/monitors/form-fields";
import {
  DNSConfigFields,
  DomainExpirationConfigFields,
  HTTPConfigFields,
  NTPConfigFields,
  PingConfigFields,
  SMTPConfigFields,
  TCPConfigFields,
  TLSConfigFields,
} from "@/components/monitors/probe-config-fields";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api-client";
import { DEFAULT_INTERVALS, MONITOR_TYPE_GROUPS, MONITOR_TYPE_ICONS } from "@/lib/monitor-meta";
import {
  buildMonitorPayload,
  buildProbeConfig,
  createMonitorFormSchema,
  defaultFormValues,
  type MonitorFormValues,
} from "@/lib/schemas";
import { cn } from "@/lib/utils";
import type { CreateMonitorInput, MonitorType } from "@/types/monitor";

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
    // Cast is safe: the schema coerces string inputs into the declared
    // output types before RHF receives the values.
    resolver: zodResolver(schema) as unknown as Resolver<MonitorFormValues>,
    defaultValues: initialValues ?? defaultFormValues("http"),
    mode: "onBlur",
  });

  const monitorType = form.watch("type");
  const watchedValues = form.watch();

  const configPreview = useMemo(
    () => JSON.stringify(buildProbeConfig(watchedValues), null, 2),
    [watchedValues],
  );

  const selectType = (nextType: MonitorType) => {
    if (typeLocked) {
      return;
    }
    form.setValue("type", nextType);
    form.setValue("interval_seconds", DEFAULT_INTERVALS[nextType]);
    form.clearErrors();
  };

  const handleSubmit = form.handleSubmit(async (values) => {
    try {
      await onSubmit(buildMonitorPayload(values));
    } catch (error) {
      if (error instanceof ApiError && error.fields) {
        for (const [field, messages] of Object.entries(error.fields)) {
          const formField = apiFieldToFormField(field);
          if (formField && messages.length > 0) {
            form.setError(formField, { type: "server", message: messages[0] });
          }
        }
      }
      throw error;
    }
  });

  return (
    <form
      onSubmit={(event) => {
        void handleSubmit(event).catch(() => undefined);
      }}
      noValidate
      className="flex flex-col gap-6"
    >
      <Card>
        <CardHeader>
          <CardTitle>{t("form.typeTitle")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {MONITOR_TYPE_GROUPS.map((group) => (
            <div key={group.key}>
              <p className="mb-2 text-xs font-medium text-muted-foreground">
                {tGroups(group.key)}
              </p>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                {group.types.map((availableType) => {
                  const Icon = MONITOR_TYPE_ICONS[availableType];
                  const selected = monitorType === availableType;

                  return (
                    <button
                      key={availableType}
                      type="button"
                      disabled={typeLocked && !selected}
                      onClick={() => selectType(availableType)}
                      aria-pressed={selected}
                      className={cn(
                        "flex items-center gap-2 rounded-lg border px-3 py-2.5 text-start text-sm transition-colors",
                        selected
                          ? "border-primary bg-primary/5 font-medium text-foreground ring-1 ring-primary"
                          : "border-border text-muted-foreground hover:border-primary/40 hover:text-foreground",
                        typeLocked && !selected && "opacity-40",
                      )}
                    >
                      <Icon className="size-4 shrink-0" aria-hidden />
                      <span className="truncate">{tTypes(availableType)}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

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
            placeholder={t(`form.targetPlaceholder.${monitorType}`)}
          />
        </CardContent>
      </Card>

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

      <Card>
        <CardHeader>
          <CardTitle>{t("form.probeConfig")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          {monitorType === "http" && <HTTPConfigFields form={form} />}
          {monitorType === "tcp" && <TCPConfigFields form={form} />}
          {monitorType === "dns" && <DNSConfigFields form={form} />}
          {monitorType === "ping" && <PingConfigFields form={form} />}
          {monitorType === "tls" && <TLSConfigFields form={form} />}
          {monitorType === "domain_expiration" && (
            <DomainExpirationConfigFields form={form} />
          )}
          {monitorType === "smtp" && <SMTPConfigFields form={form} />}
          {monitorType === "ntp" && <NTPConfigFields form={form} />}

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

      <div className="flex justify-end">
        <Button type="submit" disabled={pending} className="min-w-36">
          {submitLabel}
        </Button>
      </div>
    </form>
  );
}

// apiFieldToFormField maps backend validation field paths to form fields.
function apiFieldToFormField(field: string): keyof MonitorFormValues | null {
  const direct: Record<string, keyof MonitorFormValues> = {
    name: "name",
    target: "target",
    type: "type",
    interval_seconds: "interval_seconds",
    timeout_millis: "timeout_millis",
    retries: "retries",
    "config.port": "tcp_port",
    "config.method": "http_method",
    "config.expected_status_codes": "http_expected_status_codes",
    "config.record_type": "dns_record_type",
    "config.server": "dns_server",
    "config.mode": "smtp_mode",
    "config.ehlo_domain": "smtp_ehlo_domain",
    "config.version": "ntp_version",
  };

  return direct[field] ?? null;
}
