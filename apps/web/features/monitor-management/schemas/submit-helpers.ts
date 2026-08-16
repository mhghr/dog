import type { UseFormReturn } from "react-hook-form";

import { ApiError } from "@/shared/api";
import { getMonitorFormField } from "@/plugins/monitoring/core/registry";

import type { MonitorFormValues } from "@/features/monitor-management/schemas/schemas";
import type { MonitorType } from "@/entities/monitor/model/types";

type FormApi = Pick<UseFormReturn<MonitorFormValues>, "setError">;

// mapServerErrors copies server-side field errors onto matching form fields so
// they render alongside client-side validation.
export function mapServerErrors(form: FormApi, monitorType: MonitorType, error: unknown) {
  if (!(error instanceof ApiError) || !error.fields) {
    return;
  }

  for (const [field, messages] of Object.entries(error.fields)) {
    const formField = getMonitorFormField(monitorType, field);
    if (formField && messages.length > 0) {
      form.setError(formField, { type: "server", message: messages[0] });
    }
  }
}
