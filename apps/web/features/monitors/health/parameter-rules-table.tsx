"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { RuleBuilderSheet } from "@/features/monitors/health/rule-builder-sheet";
import { useParameterCatalog } from "@/hooks/use-health-rules";
import type {
  HealthState,
  ParameterDefinition,
  ParameterHealthState,
  ParameterRule,
} from "@/types/health";
import type { MonitorType } from "@/types/monitor";

interface ParameterRulesTableProps {
  monitorId: string;
  monitorType: MonitorType;
  rules: ParameterRule[] | undefined;
  healthStates: ParameterHealthState[] | undefined;
}

const HEALTH_BADGE_STYLES: Record<HealthState, string> = {
  OK: "border-transparent bg-success/12 text-success",
  WARNING: "border-transparent bg-warning/15 text-warning",
  ERROR: "border-transparent bg-destructive/12 text-destructive",
  UNKNOWN: "border-transparent bg-muted text-muted-foreground",
};

const HEALTH_LABEL_KEYS: Record<HealthState, string> = {
  OK: "status.up",
  WARNING: "health.warningThreshold",
  ERROR: "health.errorThreshold",
  UNKNOWN: "status.unknown",
};

function getStateFromStates(
  states: ParameterHealthState[] | undefined,
  parameterKey: string,
): HealthState {
  const state = states?.find((s) => s.parameter_key === parameterKey);
  return state?.current_state ?? "UNKNOWN";
}

function getStateValue(
  states: ParameterHealthState[] | undefined,
  parameterKey: string,
): number | undefined {
  return states?.find((s) => s.parameter_key === parameterKey)?.current_value;
}

function getRuleForParam(
  rules: ParameterRule[] | undefined,
  parameterKey: string,
): ParameterRule | undefined {
  return rules?.find((r) => r.parameter_key === parameterKey);
}

function ruleSourceLabel(
  rule: ParameterRule | undefined,
  t: ReturnType<typeof useTranslations<"health">>,
): string {
  if (!rule) return t("inherited");
  if (rule.mode === "INHERIT_DEFAULT") return t("inherited");
  if (rule.mode === "USE_PROFILE") return rule.profile ?? t("inherited");
  if (rule.mode === "DISABLED") return t("disabled");
  return t("custom");
}

export function ParameterRulesTable({
  monitorId,
  monitorType,
  rules,
  healthStates,
}: ParameterRulesTableProps) {
  const t = useTranslations("health");
  const tStatus = useTranslations();
  const catalogQuery = useParameterCatalog(monitorType);
  const parameters = catalogQuery.data ?? [];

  const [selectedParam, setSelectedParam] = useState<{
    definition: ParameterDefinition;
    rule: ParameterRule | undefined;
  } | null>(null);

  if (catalogQuery.isPending) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t("parameters")}</CardTitle>
        </CardHeader>
        <CardContent>
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="mb-2 h-10 w-full rounded-lg" />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>{t("parameters")}</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("parameters")}</TableHead>
                <TableHead>{t("currentValue")}</TableHead>
                <TableHead>{t("currentState")}</TableHead>
                <TableHead>{t("ruleSource")}</TableHead>
                <TableHead>{t("notifications")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {parameters.map((param) => {
                const rule = getRuleForParam(rules, param.key);
                const state = getStateFromStates(healthStates, param.key);
                const value = getStateValue(healthStates, param.key);

                return (
                  <TableRow
                    key={param.key}
                    className="cursor-pointer"
                    onClick={() => setSelectedParam({ definition: param, rule })}
                  >
                    <TableCell className="font-medium">
                      <div>{param.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {param.description}
                      </div>
                    </TableCell>
                    <TableCell>
                      {value !== undefined ? (
                        <span>
                          {value}
                          {param.unit ? ` ${param.unit}` : ""}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">--</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge className={HEALTH_BADGE_STYLES[state]}>
                        {tStatus(HEALTH_LABEL_KEYS[state])}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">
                        {ruleSourceLabel(rule, t)}
                      </span>
                    </TableCell>
                    <TableCell>
                      {rule && rule.mode !== "DISABLED" ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedParam({ definition: param, rule });
                          }}
                        >
                          {t("notifications")}
                        </Button>
                      ) : (
                        <span className="text-xs text-muted-foreground">--</span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {selectedParam && (
        <RuleBuilderSheet
          open={Boolean(selectedParam)}
          onOpenChange={(open) => {
            if (!open) setSelectedParam(null);
          }}
          monitorId={monitorId}
          parameter={selectedParam.definition}
          existingRule={selectedParam.rule}
          currentValue={getStateValue(healthStates, selectedParam.definition.key)}
        />
      )}
    </>
  );
}
