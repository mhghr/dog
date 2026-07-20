import { expect, test } from "@playwright/test";

// Covers the spec's end-to-end path: create monitor -> observe it in the
// list -> open details -> pause/resume -> delete.
test.describe("monitor lifecycle", () => {
  const monitorName = `E2E HTTP ${Date.now()}`;

  test("create, observe, pause, resume and delete a monitor", async ({ page }) => {
    await page.goto("/fa/app/monitors/new");

    await page.getByLabel("نام").fill(monitorName);
    await page.getByLabel("هدف").fill("https://example.com");
    await page.getByRole("button", { name: "ایجاد مانیتور" }).click();

    await expect(page).toHaveURL(/\/app\/monitors\/[0-9a-f-]+/);
    await expect(page.getByRole("heading", { name: monitorName })).toBeVisible();

    await page.goto("/fa/app/monitors");
    await expect(page.getByRole("link", { name: monitorName })).toBeVisible();

    const row = page.getByRole("row").filter({ hasText: monitorName });
    await row.getByRole("button").last().click();
    await page.getByRole("menuitem", { name: "توقف" }).click();
    await expect(row.getByText("متوقف")).toBeVisible();

    await row.getByRole("button").last().click();
    await page.getByRole("menuitem", { name: "ادامه" }).click();

    await row.getByRole("button").last().click();
    await page.getByRole("menuitem", { name: "حذف" }).click();
    await page.getByRole("button", { name: "حذف" }).click();
    await expect(page.getByRole("link", { name: monitorName })).toHaveCount(0);
  });
});
