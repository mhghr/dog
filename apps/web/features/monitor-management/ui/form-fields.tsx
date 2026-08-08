"use client";

import type { Path, UseFormReturn } from "react-hook-form";
import { Controller } from "react-hook-form";

import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";
import { Textarea } from "@/shared/ui/textarea";
import type { MonitorFormValues } from "@/features/monitor-management/schemas/schemas";

type MonitorForm = UseFormReturn<MonitorFormValues>;
type FieldName = Path<MonitorFormValues>;

function fieldError(form: MonitorForm, name: FieldName): string | undefined {
  const error = form.formState.errors[name as keyof MonitorFormValues];
  return error?.message as string | undefined;
}

function FieldShell({
  name,
  label,
  hint,
  error,
  children,
}: {
  name: string;
  label: string;
  hint?: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={name}>{label}</Label>
      {children}
      {hint && !error ? (
        <p className="text-xs text-muted-foreground">{hint}</p>
      ) : null}
      {error ? (
        <p id={`${name}-error`} role="alert" className="text-xs text-destructive">
          {error}
        </p>
      ) : null}
    </div>
  );
}

export function TextField({
  form,
  name,
  label,
  placeholder,
  hint,
  dir,
}: {
  form: MonitorForm;
  name: FieldName;
  label: string;
  placeholder?: string;
  hint?: string;
  dir?: "ltr" | "rtl";
}) {
  const error = fieldError(form, name);

  return (
    <FieldShell name={name} label={label} hint={hint} error={error}>
      <Input
        id={name}
        placeholder={placeholder}
        dir={dir}
        aria-invalid={Boolean(error)}
        aria-describedby={error ? `${name}-error` : undefined}
        {...form.register(name)}
      />
    </FieldShell>
  );
}

export function TextAreaField({
  form,
  name,
  label,
  placeholder,
  hint,
  rows = 3,
}: {
  form: MonitorForm;
  name: FieldName;
  label: string;
  placeholder?: string;
  hint?: string;
  rows?: number;
}) {
  const error = fieldError(form, name);

  return (
    <FieldShell name={name} label={label} hint={hint} error={error}>
      <Textarea
        id={name}
        dir="ltr"
        rows={rows}
        placeholder={placeholder}
        aria-invalid={Boolean(error)}
        {...form.register(name)}
      />
    </FieldShell>
  );
}

export function NumberField({
  form,
  name,
  label,
  hint,
  min,
  max,
  step,
}: {
  form: MonitorForm;
  name: FieldName;
  label: string;
  hint?: string;
  min?: number;
  max?: number;
  step?: number;
}) {
  const error = fieldError(form, name);

  return (
    <FieldShell name={name} label={label} hint={hint} error={error}>
      <Input
        id={name}
        type="number"
        inputMode="numeric"
        dir="ltr"
        min={min}
        max={max}
        step={step}
        aria-invalid={Boolean(error)}
        {...form.register(name)}
      />
    </FieldShell>
  );
}

export function SwitchField({
  form,
  name,
  label,
}: {
  form: MonitorForm;
  name: FieldName;
  label: string;
}) {
  return (
    <Controller
      control={form.control}
      name={name}
      render={({ field }) => (
        <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5">
          <Label htmlFor={name} className="cursor-pointer font-normal">
            {label}
          </Label>
          <Switch
            id={name}
            checked={Boolean(field.value)}
            onCheckedChange={field.onChange}
          />
        </div>
      )}
    />
  );
}

export function SelectField({
  form,
  name,
  label,
  options,
  hint,
}: {
  form: MonitorForm;
  name: FieldName;
  label: string;
  options: Array<{ value: string; label: string }>;
  hint?: string;
}) {
  const error = fieldError(form, name);

  return (
    <Controller
      control={form.control}
      name={name}
      render={({ field }) => (
        <FieldShell name={name} label={label} hint={hint} error={error}>
          <Select
            value={field.value === undefined ? undefined : String(field.value)}
            onValueChange={field.onChange}
          >
            <SelectTrigger id={name} aria-invalid={Boolean(error)}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FieldShell>
      )}
    />
  );
}
