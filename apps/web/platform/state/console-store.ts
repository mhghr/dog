"use client";

import { create } from "zustand";

// Client-side console UI state (sidebar collapse, mobile nav). Pure UI state
// that has no server counterpart.
interface ConsoleState {
  sidebarCollapsed: boolean;
  mobileNavOpen: boolean;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebar: () => void;
  setMobileNavOpen: (open: boolean) => void;
}

export const useConsoleStore = create<ConsoleState>((set) => ({
  sidebarCollapsed: false,
  mobileNavOpen: false,
  setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  setMobileNavOpen: (mobileNavOpen) => set({ mobileNavOpen }),
}));
