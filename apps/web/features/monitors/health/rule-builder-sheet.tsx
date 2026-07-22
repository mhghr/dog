"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { NotificationBuilder } from "@/features/monitors/health/notification-builder";
import {
  useUpdateParameterRule,
  useNotificationChannels,
  useNotificationPolicies,
  useCreateNotificationPolicy,
  useUpdateNotificationPolicy,
} from "@/hooks/use-health-rules";
import type {
  HealthState,
  NotificationPolicy,
  ParameterDefinition,
  ParameterRule,
  RuleMode,
  RuleProfile,
} from "@/types/health";
import { cn } from "@/lib/utils";

interface RuleBuilderSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  monitorId: string;
  parameter: ParameterDefinition;
  existingRule: ParameterRule | undefined;
  currentValue?: number;
}

const HEALTH_BADGE_STYLES: Record<HealthState, string> = {
  OK: "border-transparent bg-success/12 text-success",
  WARNING: "border-transparent bg-warning/15 text-warning",
  ERROR: "border-transparent bg-destructive/12 text-destructive",
  UNKNOWN: "border-transparent bg-muted text-muted-foreground",
};

function computePreviewState(
  value: number | undefined,
  warningVal: number | undefined,
  errorVal: number | undefined,
  direction: string,
): HealthState {
  if (value === undefined) return "UNKNOWN";
  if (direction === "HIGHER_IS_WORSE") {
    if (errorVal !== undefined && value >= errorVal) return "ERROR";
    if (warningVal !== undefined && value >= warningVal) return "WARNING";
    return "OK";
  }
  if (direction === "LOWER_IS_WORSE") {
    if (errorVal !== undefined && value <= errorVal) return "ERROR";
    if (warningVal !== undefined && value <= warningVal) return "WARNING";
    return "OK";
  }
  if (direction === "BOOLEAN_FAILURE") {
    return value === 0 ? "OK" : "ERROR";
  }
  return "OK";
}

export function RuleBuilderSheet({
  open,
  onOpenChange,
  monitorId,
  parameter,
  existingRule,
  currentValue,
}: RuleBuilderSheetProps) {
  const t = useTranslations("health");
  const tStatus = useTranslations();

  const [mode, setMode] = useState<RuleMode>(
    existingRule?.mode ?? "INHERIT_DEFAULT",
  );
  const [profile, setProfile] = useState<RuleProfile>(
    existingRule?.profile ?? parameter.default_profile,
  );
  const [warningValue, setWarningValue] = useState<number | undefined>(
    existingRule?.warning_value,
  );
  const [errorValue, setErrorValue] = useState<number | undefined>(
    existingRule?.error_value,
  );
  const [recoveryValue, setRecoveryValue] = useState<number | undefined>(
    existingRule?.recovery_value,
  );
  const [enabled, setEnabled] = useState(existingRule?.enabled ?? true);

  const updateRule = useUpdateParameterRule(monitorId);
  const channelsQuery = useNotificationChannels();
  const policiesQuery = useNotificationPolicies(monitorId);
  const createPolicy = useCreateNotificationPolicy();
  const updatePolicy = useUpdateNotificationPolicy();

  const paramPolicies =
    policiesQuery.data?.filter(
      (p) => p.parameter_key === parameter.key,
    ) ?? [];

  const previewState = computePreviewState(
    currentValue,
    warningValue,
    errorValue,
    parameter.direction,
  );

  const handleSave = async () => {
    try {
      await updateRule.mutateAsync({
        monitor_id: monitorId,
        parameter_key: parameter.key,
        mode,
        profile: mode === "USE_PROFILE" ? profile : undefined,
        warning_value: mode === "CUSTOM" ? warningValue : undefined,
        warning_operator: mode === "CUSTOM" ? "gt" : "",
        error_value: mode === "CUSTOM" ? errorValue : undefined,
        error_operator: mode === "CUSTOM" ? "gt" : "",
        recovery_value: mode === "CUSTOM" ? recoveryValue : undefined,
        recovery_operator: mode === "CUSTOM" ? "lt" : undefined,
        aggregation: "avg",
        window_type: "rolling",
        window_value: 300,
        minimum_samples: 3,
        missing_data_policy: "ignore",
        missed_checks: 10,
        cooldown_seconds: 300,
        enabled,
      });
      toast.success(tStatus("validation.updateSuccess"));
      onOpenChange(false);
    } catch {
      toast.error(tStatus("validation.genericError"));
    }
  };

  const handleReset = () => {
    setMode("INHERIT_DEFAULT");
    setProfile(parameter.default_profile);
    setWarningValue(undefined);
    setErrorValue(undefined);
    setRecoveryValue(undefined);
    setEnabled(true);
  };

  const handleTestNotification = () => {
    toast.success(t("testNotification"));
  };

  const handleAddPolicy = async () => {
    const firstChannel = channelsQuery.data?.[0];
    if (!firstChannel) return;

    try {
      await createPolicy.mutateAsync({
        monitor_id: monitorId,
        parameter_key: parameter.key,
        channel_id: firstChannel.id,
        triggers: ["STATUS_ENTERED_WARNING", "STATUS_ENTERED_ERROR"],
        delay_seconds: 0,
        repeat_interval_seconds: 300,
        cooldown_seconds: 600,
        enabled: true,
      });
    } catch {
      toast.error(tStatus("validation.genericError"));
    }
  };

  const handleUpdatePolicy = (policy: NotificationPolicy) => {
    if (!policy.id) return;
    updatePolicy.mutate({ ...policy, id: policy.id });
  };

  const handleRemovePolicy = (policyId: string) => {
    updatePolicy.mutate({
      id: policyId,
      monitor_id: monitorId,
      parameter_key: parameter.key,
      channel_id: "",
      triggers: [],
      delay_seconds: 0,
      repeat_interval_seconds: 0,
      cooldown_seconds: 0,
      enabled: false,
    });
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-full sm:max-w-lg overflow-y-auto"
      >
        <SheetHeader>
          <SheetTitle>
            {t("ruleBuilder")}: {parameter.name}
          </SheetTitle>
          <SheetDescription>{parameter.description}</SheetDescription>
        </SheetHeader>

        <div className="flex flex-col gap-5 px-4">
          <div className="flex flex-col gap-2">
            <Label>{t("ruleSource")}</Label>
            <Select
              value={mode}
              onValueChange={(v) => setMode(v as RuleMode)}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="INHERIT_DEFAULT">{t("inherited")}</SelectItem>
                <SelectItem value="USE_PROFILE">
                  {profile}
                </SelectItem>
                <SelectItem value="CUSTOM">{t("custom")}</SelectItem>
                <SelectItem value="DISABLED">{t("disabled")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {mode === "USE_PROFILE" && (
            <div className="flex flex-col gap-2">
              <Label>{t("ruleSource")}</Label>
              <Select
                value={profile}
                onValueChange={(v) => setProfile(v as RuleProfile)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Sensitive">Sensitive</SelectItem>
                  <SelectItem value="Recommended">Recommended</SelectItem>
                  <SelectItem value="Relaxed">Relaxed</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          {mode === "CUSTOM" && (
            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-2">
                <Label>
                  {t("warningThreshold")} {parameter.unit ? `(${parameter.unit})` : ""}
                </Label>
                <Input
                  type="number"
                  value={warningValue ?? ""}
                  onChange={(e) =>
                    setWarningValue(
                      e.target.value ? Number(e.target.value) : undefined,
                    )
                  }
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label>
                  {t("errorThreshold")} {parameter.unit ? `(${parameter.unit})` : ""}
                </Label>
                <Input
                  type="number"
                  value={errorValue ?? ""}
                  onChange={(e) =>
                    setErrorValue(
                      e.target.value ? Number(e.target.value) : undefined,
                    )
                  }
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label>
                  {t("recoveryThreshold")} {parameter.unit ? `(${parameter.unit})` : ""}
                </Label>
                <Input
                  type="number"
                  value={recoveryValue ?? ""}
                  onChange={(e) =>
                    setRecoveryValue(
                      e.target.value ? Number(e.target.value) : undefined,
                    )
                  }
                />
              </div>
            </div>
          )}

          <div className="flex items-center justify-between rounded-lg border border-border p-3">
            <Label>{t("preview")}</Label>
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">
                {t("currentValue")}:{" "}
                {currentValue !== undefined
                  ? `${currentValue}${parameter.unit ? ` ${parameter.unit}` : ""}`
                  : "--"}
              </span>
              <Badge className={cn(HEALTH_BADGE_STYLES[previewState])}>
                {tStatus(
                  previewState === "OK"
                    ? "status.up"
                    : previewState === "WARNING"
                      ? "health.warningThreshold"
                      : previewState === "ERROR"
                        ? "health.errorThreshold"
                        : "status.unknown",
                )}
              </Badge>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <Label>{tStatus("monitors.fields.enabled")}</Label>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>

          <div className="space-y-2">
            <Label className="text-sm font-medium">{t("notifications")}</Label>
            <NotificationBuilder
              policies={paramPolicies}
              channels={channelsQuery.data ?? []}
              onAdd={handleAddPolicy}
              onUpdate={handleUpdatePolicy}
              onRemove={handleRemovePolicy}
            />
          </div>
        </div>

        <SheetFooter className="flex-row gap-2 pt-2">
          <Button variant="outline" size="sm" onClick={handleReset}>
            {t("resetToDefault")}
          </Button>
          <Button variant="outline" size="sm" onClick={handleTestNotification}>
            {t("testNotification")}
          </Button>
          <Button
            size="sm"
            onClick={handleSave}
            disabled={updateRule.isPending}
          >
            {tStatus("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
