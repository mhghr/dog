"use client";

import { useState } from "react";
import { Info, Plus, Trash2 } from "lucide-react";

import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Switch } from "@/shared/ui/switch";
import { Textarea } from "@/shared/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select";
import type { SchemaField } from "../monitoring-schema";

interface SchemaFieldProps {
  field: SchemaField;
  value: unknown;
  isFa: boolean;
  error?: string;
  onChange: (value: unknown) => void;
}

// Info icon that reveals the field description in a tooltip instead of
// cluttering the form with inline helper text.
function FieldHint({ text }: { text: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex cursor-help text-muted-foreground/40 transition-colors hover:text-muted-foreground">
          <Info className="size-3.5" />
        </span>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-60 text-xs">
        {text}
      </TooltipContent>
    </Tooltip>
  );
}

// Key/value pair editor for header-like fields. Values support `${secret:name}`
// references; raw secret values are never stored in the form state.
function KeyValueEditor({
  value,
  onChange,
}: {
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const [rows, setRows] = useState<Array<{ key: string; value: string }>>(() => {
    const record = (value ?? {}) as Record<string, string>;
    const entries = Object.entries(record);
    return entries.length > 0 ? entries.map(([key, val]) => ({ key, value: val })) : [{ key: "", value: "" }];
  });

  const update = (rows: Array<{ key: string; value: string }>) => {
    setRows(rows);
    const record: Record<string, string> = {};
    for (const row of rows) {
      if (row.key.trim() !== "") {
        record[row.key.trim()] = row.value;
      }
    }
    onChange(record);
  };

  const setRow = (index: number, key: string, nextValue: string) => {
    const next = rows.map((row, i) => (i === index ? { ...row, [key]: nextValue } : row));
    update(next);
  };

  const addRow = () => update([...rows, { key: "", value: "" }]);
  const removeRow = (index: number) => update(rows.filter((_, i) => i !== index));

  return (
    <div className="flex flex-col gap-2">
      {/* Three columns below each other: [Add Header] [Key] [Value]. Forced LTR
          so the Add Header button stays on the left even in RTL layouts. */}
      <div dir="ltr" className="grid grid-cols-[auto_minmax(0,1fr)_minmax(0,2fr)] items-start gap-2">
        {/* Add Header button — first column, its own row. */}
        <button
          type="button"
          onClick={addRow}
          className="inline-flex h-10 items-center justify-center gap-1.5 rounded-lg border border-dashed border-border px-3 text-xs text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
        >
          <Plus className="size-4" />
          Add header
        </button>

        {/* Key inputs column */}
        <div className="flex flex-col gap-2">
          {rows.map((row, index) => (
            <Input
              key={`k-${index}`}
              value={row.key}
              placeholder="Header"
              dir="ltr"
              className="h-10 font-mono text-xs"
              onChange={(e) => setRow(index, "key", e.target.value)}
            />
          ))}
        </div>

        {/* Value inputs column, delete button at the end of each row */}
        <div className="flex flex-col gap-2">
          {rows.map((row, index) => (
            <div key={`v-${index}`} className="flex items-center gap-2">
              <Input
                value={row.value}
                placeholder="Value or ${secret:name}"
                dir="ltr"
                className="h-10 flex-1 font-mono text-xs"
                onChange={(e) => setRow(index, "value", e.target.value)}
              />
              <button
                type="button"
                onClick={() => removeRow(index)}
                aria-label="Remove header"
                className="grid size-10 shrink-0 place-items-center rounded-lg text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
              >
                <Trash2 className="size-4" />
              </button>
            </div>
          ))}
        </div>
      </div>

      {rows.length === 0 && (
        <p className="text-xs text-muted-foreground">No headers yet</p>
      )}
    </div>
  );
}

// Renders a single schema-driven field. The widget is declared by the
// monitoring type schema; no monitoring type hardcodes its own field UI.
// A switch field rendered with its label above the toggle, matching the input
// boxes' label-above-control layout.
export function SchemaToggleRow({ field, value, isFa, onChange }: Omit<SchemaFieldProps, "error">) {
  const label = isFa ? field.label.fa : field.label.en;
  const help = field.help ? (isFa ? field.help.fa : field.help.en) : undefined;

  return (
    <div className="flex flex-col gap-2">
      <Label className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
        {label}
        {help && <FieldHint text={help} />}
      </Label>
      <Switch checked={Boolean(value)} onCheckedChange={onChange} />
    </div>
  );
}

export function SchemaFieldRenderer({ field, value, isFa, error, onChange }: SchemaFieldProps) {
  const label = isFa ? field.label.fa : field.label.en;
  const help = field.help ? (isFa ? field.help.fa : field.help.en) : undefined;

  return (
    <div className="flex flex-col gap-2">
      <Label className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground" data-invalid={error ? "true" : undefined}>
        {label}
        {help && <FieldHint text={help} />}
      </Label>

      {field.widget === "switch" ? (
        <Switch checked={Boolean(value)} onCheckedChange={onChange} />
      ) : field.widget === "select" ? (
        <Select
          value={String(value ?? field.defaultValue ?? "")}
          onValueChange={onChange}
        >
          <SelectTrigger className="w-full" style={{ height: "2.5rem" }}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {field.options?.map((option) => (
              <SelectItem key={option} value={option}>{option}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : field.widget === "textarea" ? (
        <Textarea
          value={typeof value === "string" ? value : String(field.defaultValue ?? "")}
          onChange={(e) => onChange(e.target.value)}
          dir="ltr"
          rows={3}
        />
      ) : field.widget === "keyvalue" ? (
        <KeyValueEditor value={value} onChange={onChange} />
      ) : field.widget === "number" ? (
        <Input
          type="number"
          value={typeof value === "number" ? value : Number(field.defaultValue ?? 0)}
          min={field.min}
          max={field.max}
          step={field.step}
          className="h-10"
          dir="ltr"
          data-invalid={error ? "true" : undefined}
          onChange={(e) => onChange(Number(e.target.value))}
        />
      ) : (
        <Input
          type="text"
          value={typeof value === "string" ? value : String(field.defaultValue ?? "")}
          placeholder={field.placeholder}
          className="h-10"
          dir="ltr"
          data-invalid={error ? "true" : undefined}
          onChange={(e) => onChange(e.target.value)}
        />
      )}

      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  );
}
