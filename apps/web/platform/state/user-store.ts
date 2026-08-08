"use client";

import { create } from "zustand";

// Client-side user identity for UI rendering (avatar, menu). The authoritative
// session record stays in React Query via useMe(); this store only mirrors the
// minimal identity needed outside query scope.
interface UserState {
  id: string | null;
  name: string;
  email: string;
  avatarUrl: string;
  roles: string[];
  setUser: (user: Partial<UserState>) => void;
  clear: () => void;
}

export const useUserStore = create<UserState>((set) => ({
  id: null,
  name: "",
  email: "",
  avatarUrl: "",
  roles: [],
  setUser: (user) => set((state) => ({ ...state, ...user })),
  clear: () =>
    set({ id: null, name: "", email: "", avatarUrl: "", roles: [] }),
}));
