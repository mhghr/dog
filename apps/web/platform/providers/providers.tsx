"use client";

import { useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { useLocale } from "next-intl";
import { ThemeProvider } from "next-themes";
import { Direction } from "radix-ui";

import { createQueryClient } from "@/shared/data/query-client";
import { Toaster } from "@/shared/ui/sonner";
import { TooltipProvider } from "@/shared/ui/tooltip";

export function AppProviders({ children }: { children: React.ReactNode }) {
  const locale = useLocale();
  const [queryClient] = useState(() => createQueryClient());

  return (
    <Direction.Provider dir={locale === "fa" ? "rtl" : "ltr"}>
      <ThemeProvider
        attribute="class"
        defaultTheme="system"
        enableSystem
        disableTransitionOnChange
      >
        <QueryClientProvider client={queryClient}>
          <TooltipProvider delayDuration={200}>{children}</TooltipProvider>
          <Toaster position="bottom-center" />
        </QueryClientProvider>
      </ThemeProvider>
    </Direction.Provider>
  );
}
