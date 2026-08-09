"use client";

import { useTranslations } from "next-intl";
import { useState } from "react";
import { Check, ChevronsUpDown, Plus } from "lucide-react";

import { Button } from "@/shared/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { useWorkspaces } from "@/entities/workspace/hooks/use-workspace";
import { useWorkspace } from "@/widgets/console-shell/workspace-provider";
import { useRouter } from "@/i18n/navigation";

export function WorkspaceSwitcher() {
  const t = useTranslations("navigation");
  const router = useRouter();
  const { slug: activeSlug } = useWorkspace();
  const { data: workspacesData } = useWorkspaces();
  const [open, setOpen] = useState(false);

  const workspaces = workspacesData?.items ?? [];
  const activeWorkspace = workspaces.find((w) => w.slug === activeSlug);
  const hasMultiple = workspaces.length > 1;

  const handleSelect = (slug: string) => {
    setOpen(false);
    router.replace(`/console/w/${slug}/dashboard`);
  };

  // Single workspace: show name only, no dropdown
  if (!hasMultiple) {
    return (
      <div className="flex items-center gap-2 px-2">
        <span className="grid size-6 shrink-0 place-items-center rounded-md bg-primary/10 text-xs font-bold text-primary">
          {activeWorkspace?.name?.charAt(0)?.toUpperCase() ?? "W"}
        </span>
        <span className="truncate text-sm font-medium text-foreground">
          {activeWorkspace?.name ?? t("workspaceDefault")}
        </span>
      </div>
    );
  }

  // Multiple workspaces: dropdown switcher
  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="max-w-44 gap-1 px-2 data-[state=open]:bg-muted"
        >
          <span className="grid size-5 shrink-0 place-items-center rounded bg-primary/10 text-[10px] font-bold text-primary">
            {activeWorkspace?.name?.charAt(0)?.toUpperCase() ?? "W"}
          </span>
          <span className="truncate text-xs font-medium">
            {activeWorkspace?.name ?? t("workspaceDefault")}
          </span>
          <ChevronsUpDown className="size-3 shrink-0 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-52">
        <DropdownMenuLabel className="text-xs text-muted-foreground">
          {t("workspaces")}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {workspaces.map((ws) => (
          <DropdownMenuItem
            key={ws.id}
            onSelect={() => handleSelect(ws.slug)}
            className="flex items-center justify-between gap-2"
          >
            <span className="flex min-w-0 items-center gap-2">
              <span className="grid size-6 shrink-0 place-items-center rounded-md bg-primary/10 text-xs font-bold text-primary">
                {ws.name.charAt(0).toUpperCase()}
              </span>
              <span className="truncate text-sm">{ws.name}</span>
            </span>
            {ws.slug === activeSlug ? (
              <Check className="size-4 shrink-0 text-primary" />
            ) : null}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onSelect={() => router.replace("/app/resources")}
        >
          <Plus className="size-4" />
          <span className="text-sm">{t("createWorkspace")}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
