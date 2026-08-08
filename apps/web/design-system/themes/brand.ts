import type { ThemeTokens } from "./default";
import { defaultTheme } from "./default";

// White-label theme. Branded tenants override the default palette without
// touching the token structure. Every field is optional — missing values
// fall through to the default theme.
export type BrandTheme = DeepPartial<ThemeTokens>;

type DeepPartial<T> = {
  [K in keyof T]?: T[K] extends object ? DeepPartial<T[K]> : T[K];
};

export function resolveTheme(overrides?: BrandTheme): ThemeTokens {
  if (!overrides) {
    return defaultTheme;
  }
  return {
    ...defaultTheme,
    ...overrides,
    colors: { ...defaultTheme.colors, ...overrides.colors },
    radius: { ...defaultTheme.radius, ...overrides.radius },
    font: { ...defaultTheme.font, ...overrides.font },
  };
}

export const brandThemes: Record<string, BrandTheme> = {
  default: {},
};
