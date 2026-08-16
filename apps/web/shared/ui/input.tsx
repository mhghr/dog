import * as React from "react"

import { cn } from "@/shared/utils/cn"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-10 w-full min-w-0 rounded-lg border border-border/70 bg-white px-3 py-2 text-base text-foreground transition-[border-color,box-shadow] duration-150 outline-none placeholder:text-muted-foreground/70 focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-ring/15 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-muted/40 disabled:opacity-50 aria-invalid:border-destructive/60 aria-invalid:ring-destructive/10 md:text-sm dark:border-border/50 dark:bg-background",
        className
      )}
      {...props}
    />
  )
}

export { Input }
