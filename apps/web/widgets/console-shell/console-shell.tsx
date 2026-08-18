"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useLocale, useTranslations } from "next-intl";

import { BrandMark } from "@/shared/ui/brand-mark";
import { ConsoleBreadcrumbs } from "@/widgets/console-shell/console-breadcrumbs";
import { LanguageSwitcher } from "@/shared/ui/language-switcher";
import { ThemeToggle } from "@/shared/ui/theme-toggle";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import type { AuthUser } from "@/shared/types/auth";
import { useAuth } from "@/platform/auth/auth-provider";
import { useLogout } from "@/platform/auth/use-auth";
import { useLiveResults } from "@/platform/realtime/use-live-results";
import { Link, usePathname, useRouter } from "@/i18n/navigation";
import { cn } from "@/shared/utils/cn";
import {
  Broadcast,
  Browser,
  GearSix,
  Monitor,
  SignOut,
  SquaresFour,
  UserCircle,
  Warning,
} from "@/shared/ui/icons";

type SidebarState = {
  collapsed: boolean;
  toggle: () => void;
  mobileOpen: boolean;
  setMobileOpen: (v: boolean) => void;
};

const SidebarCtx = createContext<SidebarState | null>(null);

function useSidebarCtx() {
  const ctx = useContext(SidebarCtx);
  if (!ctx) throw new Error("useSidebarCtx must be used within ConsoleShell");
  return ctx;
}

const NAV_ITEMS = {
  primary: [
    { href: "/dashboard",   labelKey: "dashboard" as const,   icon: SquaresFour },
    { href: "/resources",   labelKey: "resources" as const,   icon: Monitor },
    { href: "/alerts",      labelKey: "alerts" as const,      icon: Warning },
    { href: "/status-pages",labelKey: "statusPages" as const, icon: Browser },
    { href: "/probes",      labelKey: "probes" as const,      icon: Broadcast },
  ],
  workspace: [
    { href: "/settings/members",     labelKey: "members" as const,      icon: GearSix },
    { href: "/settings/api-keys",    labelKey: "apiKeys" as const,      icon: GearSix },
    { href: "/settings/integrations",labelKey: "integrations" as const, icon: GearSix },
    { href: "/settings/billing",     labelKey: "billing" as const,      icon: GearSix },
    { href: "/settings",             labelKey: "workspaceSettings" as const, icon: GearSix },
  ],
};

function ToggleIcon({ collapsed }: { collapsed: boolean }) {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
      <line
        x1={collapsed ? "9" : "15"}
        y1="3"
        x2={collapsed ? "9" : "15"}
        y2="21"
      />
    </svg>
  );
}

function AuthGate({
  isLoaded,
  isSignedIn,
}: {
  isLoaded: boolean;
  isSignedIn: boolean;
}) {
  const router = useRouter();
  useLiveResults(isSignedIn);
  useEffect(() => {
    if (isLoaded && !isSignedIn) {
      router.replace("/login");
    }
  }, [isLoaded, isSignedIn, router]);
  return null;
}

function UserMenuComp({ user: userProp }: { user: AuthUser | null }) {
  const tAuth = useTranslations("auth");
  const router = useRouter();
  const logoutMutation = useLogout();
  const user = userProp;
  const displayName = user?.name || user?.email || user?.phone || "";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={displayName || "user"}
          className="relative shrink-0 cursor-pointer rounded-full outline-none transition-shadow focus-visible:ring-3 focus-visible:ring-ring/50 dark:hover:shadow-[0_0_12px_-2px_var(--primary)_/_30%]"
        >
          {user?.avatar_url ? (
            <img
              src={user.avatar_url}
              alt=""
              referrerPolicy="no-referrer"
              className="size-7 rounded-full ring-1 ring-border/60 dark:ring-primary/25"
            />
          ) : (
            <span className="grid size-7 place-items-center rounded-full bg-muted ring-1 ring-border/60 dark:ring-primary/25">
              <UserCircle className="size-4 text-muted-foreground" />
            </span>
          )}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-48">
        {displayName ? (
          <>
            <DropdownMenuLabel className="max-w-56 truncate" dir="auto">
              {displayName}
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
          </>
        ) : null}
        <DropdownMenuItem
          variant="destructive"
          onSelect={() => {
            logoutMutation
              .mutateAsync()
              .finally(() => router.replace("/login"));
          }}
          disabled={logoutMutation.isPending}
        >
          <SignOut className="size-4" />
          {tAuth("logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function SidebarContent({
  collapsed,
  onNav,
  toggle,
}: {
  collapsed: boolean;
  onNav?: () => void;
  toggle?: () => void;
}) {
  const t = useTranslations("navigation");
  const pathname = usePathname();

  // Workspace-scoped console lives under /console/w/[slug]/... Derive the
  // base prefix so nav links stay inside the active workspace.
  const wsMatch = pathname.match(/^\/console\/w\/([^/]+)/);
  const base = wsMatch ? `/console/w/${wsMatch[1]}` : "/app";

  const buildNav = (items: readonly { href: string; labelKey: string; icon: React.ElementType }[]) =>
    items.map((item) => ({
      ...item,
      href: `${base}${item.href}`,
    }));

  return (
    <>
      <div className="flex h-14 shrink-0 items-center justify-between border-b border-border px-3">
        <Link
          href={`${base}/dashboard`}
          onClick={onNav}
          className="flex items-center gap-3 min-w-0"
        >
          <BrandMark />
          {!collapsed && (
            <span className="truncate text-sm font-medium text-foreground">
              {t("dashboard")}
            </span>
          )}
        </Link>
        {toggle && (
          <button
            type="button"
            onClick={toggle}
            className="flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            title={collapsed ? "Expand" : "Collapse"}
          >
            <ToggleIcon collapsed={collapsed} />
          </button>
        )}
      </div>
      <nav className="flex flex-1 flex-col overflow-auto p-2 pt-3">
        {buildNav(NAV_ITEMS.primary).map((item) => {
          const Icon = item.icon;
          const active =
            pathname === item.href || pathname.startsWith(`${item.href}/`);

          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={onNav}
              data-active={active}
              className={cn(
                "group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-[background-color,color] duration-150 hover:bg-accent hover:text-foreground",
                active && "bg-accent text-foreground",
                collapsed && "justify-center px-0",
              )}
              title={collapsed ? t(item.labelKey) : undefined}
            >
              <span
                className={cn(
                  "flex size-7 shrink-0 items-center justify-center rounded-lg transition-[background-color,color] duration-150",
                  active
                    ? "bg-primary/10 text-primary group-hover:text-primary"
                    : "bg-transparent text-muted-foreground group-hover:text-foreground",
                )}
              >
                <Icon className="size-4" aria-hidden />
              </span>
              {!collapsed && (
                <span className="min-w-0 flex-1 truncate">
                  {t(item.labelKey)}
                </span>
              )}
            </Link>
          );
        })}

        <div className="my-3 border-t border-border/60" />

        {buildNav(NAV_ITEMS.workspace).map((item) => {
          const Icon = item.icon;
          const active =
            pathname === item.href || pathname.startsWith(`${item.href}/`);

          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={onNav}
              data-active={active}
              className={cn(
                "group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-[background-color,color] duration-150 hover:bg-accent hover:text-foreground",
                active && "bg-accent text-foreground",
                collapsed && "justify-center px-0",
              )}
              title={collapsed ? t(item.labelKey) : undefined}
            >
              <span
                className={cn(
                  "flex size-7 shrink-0 items-center justify-center rounded-lg transition-[background-color,color] duration-150",
                  active
                    ? "bg-primary/10 text-primary group-hover:text-primary"
                    : "bg-transparent text-muted-foreground group-hover:text-foreground",
                )}
              >
                <Icon className="size-4" aria-hidden />
              </span>
              {!collapsed && (
                <span className="min-w-0 flex-1 truncate">
                  {t(item.labelKey)}
                </span>
              )}
            </Link>
          );
        })}
      </nav>
    </>
  );
}

export function ConsoleShell({ children }: { children: ReactNode }) {
  const locale = useLocale();
  const isRtl = locale === "fa";
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const { isLoaded, isSignedIn, user } = useAuth();

  const toggle = useCallback(() => setCollapsed((c) => !c), []);
  const sidebarCtx = useMemo(
    () => ({ collapsed, toggle, mobileOpen, setMobileOpen }),
    [collapsed, toggle, mobileOpen],
  );

  return (
    <SidebarCtx.Provider value={sidebarCtx}>
      <div
        className="relative flex min-h-screen"
        dir={isRtl ? "rtl" : "ltr"}
        style={{ scrollbarGutter: "auto" }}
      >
        <AuthGate isLoaded={isLoaded} isSignedIn={isSignedIn} />

        <aside
          style={{ fontFamily: "var(--font-bakh), var(--font-estedad), ui-sans-serif, system-ui, sans-serif", fontWeight: 500 }}
          className={cn(
            "hidden shrink-0 flex-col border-e border-[var(--sidebar-border)] bg-sidebar dark:bg-transparent lg:flex",
            collapsed ? "w-[4.5rem]" : "w-64",
          )}
        >
          <SidebarContent collapsed={collapsed} />
        </aside>

        {mobileOpen && (
          <div className="fixed inset-0 z-overlay lg:hidden">
            <div
              className="absolute inset-0 bg-black/50"
              onClick={() => setMobileOpen(false)}
            />
            <aside
              className={cn(
                "absolute inset-y-0 flex w-64 flex-col bg-sidebar shadow-modal",
                isRtl ? "right-0" : "left-0",
              )}
            >
              <SidebarContent
                collapsed={false}
                onNav={() => setMobileOpen(false)}
              />
            </aside>
          </div>
        )}

        <div
          className={cn(
            "relative flex min-w-0 flex-1 flex-col",
          )}
        >
          <header className="sticky top-0 z-sticky flex h-14 items-center gap-2 border-b border-[var(--header-border)] bg-[var(--header-background)] px-4 backdrop-blur-xl">
            <button
              type="button"
              onClick={() => setMobileOpen(true)}
              className="flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground lg:hidden"
              title="Open menu"
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              >
                <line x1="4" y1="6" x2="20" y2="6" />
                <line x1="4" y1="12" x2="20" y2="12" />
                <line x1="4" y1="18" x2="20" y2="18" />
              </svg>
            </button>

            {/* Desktop collapse toggle placed next to breadcrumbs */}
            <div className="hidden lg:flex">
              <button
                type="button"
                onClick={toggle}
                className="mr-2 flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                title={collapsed ? "Expand" : "Collapse"}
              >
                <ToggleIcon collapsed={collapsed} />
              </button>
            </div>
            <ConsoleBreadcrumbs />

            <div className="flex-1" />

            <div className="flex items-center gap-1">
              <LanguageSwitcher />
              <span className="mx-1 h-5 w-px bg-border/70" aria-hidden />
              <ThemeToggle />
              <span className="mx-1 h-5 w-px bg-border/70" aria-hidden />
              <UserMenuComp user={user} />
            </div>
          </header>

          <main className="mx-auto w-full max-w-[1440px] flex-1 overflow-auto px-6 py-6 lg:px-10">
            {children}
          </main>
        </div>
      </div>
    </SidebarCtx.Provider>
  );
}
