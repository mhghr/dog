"use client";

import type { UseFormReturn } from "react-hook-form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface LocationFormValues {
  name: string;
  code: string;
}

interface LocationFormFieldsProps {
  form: UseFormReturn<LocationFormValues>;
}

export function LocationFormFields({ form }: LocationFormFieldsProps) {
  const { errors } = form.formState;

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="loc-name">Name</Label>
        <Input id="loc-name" {...form.register("name")} />
        {errors.name && (
          <p className="text-xs text-destructive">{errors.name.message}</p>
        )}
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="loc-code">Code</Label>
        <Input id="loc-code" dir="ltr" {...form.register("code")} />
        {errors.code && (
          <p className="text-xs text-destructive">{errors.code.message}</p>
        )}
      </div>
    </>
  );
}
