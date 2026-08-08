"use client";

import { create } from "zustand";

// Client-side workspace state only. Server data (workspace records, members)
// belongs in React Query; this store holds the *selected* workspace identity
// used to scope every query key and API call.
interface WorkspaceState {
  slug: string | null;
  id: string | null;
  setWorkspace: (slug: string | null, id: string | null) => void;
  clear: () => void;
}

export const useWorkspaceStore = create<WorkspaceState>((set) => ({
  slug: null,
  id: null,
  setWorkspace: (slug, id) => set({ slug, id }),
  clear: () => set({ slug: null, id: null }),
}));
