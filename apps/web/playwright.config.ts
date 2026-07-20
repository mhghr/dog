import { defineConfig } from "@playwright/test";

// E2E tests require the full stack (docker compose up) plus `pnpm dev`.
export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:2000",
    trace: "on-first-retry",
  },
});
