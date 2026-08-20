"use client";

import type { HealthRuleDef } from "../monitoring-schema";
import { localizeUnit } from "../monitoring-schema";
import { HealthSlider } from "./HealthSlider";

export type HealthRuleThreshold = { warning: number; critical: number };
export type HealthRulesState = Record<string, HealthRuleThreshold>;

function computeSliderMax(def: HealthRuleDef | undefined, fallback: number): number {
  const defaults = def?.defaults;
  const base = Math.max(defaults?.warning ?? 0, defaults?.critical ?? 0) || fallback;
  if (base > 0 && base <= 20) return Math.ceil(base * 3);
  if (base <= 100) return 100;
  return Math.max(Math.ceil(base * 2.5), 100);
}

interface HealthRulesBuilderProps {
  rules: HealthRuleDef[];
  state: HealthRulesState;
  isFa: boolean;
  onChange: (key: string, next: HealthRuleThreshold) => void;
}

// Health Rules Builder: renders the configurable (threshold) rules a
// monitoring type exposes, each as a dual-thumb slider. Boolean rules
// (availability, assertions) are always-on platform behavior and need no UI.
export function HealthRulesBuilder({ rules, state, isFa, onChange }: HealthRulesBuilderProps) {
  const thresholdRules = rules.filter((rule) => rule.direction !== "boolean");

  if (thresholdRules.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-3">
      {thresholdRules.map((rule) => {
        const value = state[rule.key] ?? {
          warning: rule.defaults?.warning ?? 0,
          critical: rule.defaults?.critical ?? 0,
        };
        const sliderMax = computeSliderMax(rule, 100);
        const step = sliderMax > 500 ? 5 : sliderMax > 100 ? 1 : 0.5;

        return (
          <div
            key={rule.key}
            className="rounded-xl border border-border/60 bg-card/60 p-4 shadow-subtle transition-[border-color,box-shadow] duration-200 hover:border-border/80 hover:shadow-card-hover"
          >
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-foreground">
                {isFa ? rule.label.fa : rule.label.en}
                {rule.unit && (
                  <span className="text-xs font-normal text-muted-foreground">
                    {" "}({localizeUnit(rule.unit, isFa)})
                  </span>
                )}
              </span>
            </div>
            <div className="mt-3">
              <HealthSlider
                value={{ warning: value.warning, critical: value.critical }}
                max={sliderMax}
                step={step}
                unit={rule.unit}
                direction={rule.direction}
                showReadout
                isFa={isFa}
                onValueChange={(warning, critical) =>
                  onChange(rule.key, { warning, critical })
                }
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}
