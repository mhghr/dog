"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { Check, Folder, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { useProjectContext } from "@/components/project-context";
import { useCreateProject } from "@/hooks/use-organization";
import { cn } from "@/lib/utils";

export function ProjectSwitcher() {
  const t = useTranslations("organization");
  const tCommon = useTranslations("common");
  const { projectId, setProjectId, projects, isLoading } = useProjectContext();
  const createMutation = useCreateProject();
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [newName, setNewName] = React.useState("");
  const inputRef = React.useRef<HTMLInputElement>(null);

  const currentProject = projects.find((p) => p.id === projectId);

  const handleCreate = async () => {
    const trimmed = newName.trim();
    if (!trimmed) return;
    try {
      const project = await createMutation.mutateAsync(trimmed);
      setProjectId(project.id);
      setNewName("");
      setDialogOpen(false);
    } catch {
      // error toast is shown by the mutation
    }
  };

  const handleDialogOpen = React.useCallback((open: boolean) => {
    setDialogOpen(open);
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 100);
    } else {
      setNewName("");
    }
  }, []);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="flex items-center gap-1.5 rounded-lg border border-border/70 bg-muted/50 px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            <Folder className="size-3" />
            <span className="max-w-32 truncate">
              {isLoading
                ? tCommon("loading")
                : currentProject?.name ?? t("noProjects")}
            </span>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="min-w-48">
          <DropdownMenuLabel>{t("projects")}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {projects.length === 0 ? (
            <div className="px-1.5 py-2 text-xs text-muted-foreground">
              {t("noProjects")}
            </div>
          ) : (
            <DropdownMenuGroup>
              {projects.map((project) => (
                <DropdownMenuItem
                  key={project.id}
                  onSelect={() => setProjectId(project.id)}
                >
                  <Folder className="size-4" />
                  <span className="flex-1 truncate">{project.name}</span>
                  {project.id === projectId && (
                    <Check className="size-4 text-primary" />
                  )}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onSelect={() => handleDialogOpen(true)}
            className={cn(
              "text-muted-foreground",
              "focus:text-foreground",
            )}
          >
            <Plus className="size-4" />
            {t("newProject")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={dialogOpen} onOpenChange={handleDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("createProject")}</DialogTitle>
            <DialogDescription>{t("projectNamePlaceholder")}</DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void handleCreate();
            }}
          >
            <Input
              ref={inputRef}
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder={t("projectNamePlaceholder")}
              maxLength={200}
            />
            <DialogFooter className="mt-4">
              <Button
                type="submit"
                disabled={!newName.trim() || createMutation.isPending}
              >
                {t("createProject")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
