// Design token theme definitions (TypeScript view of the CSS tokens in
// design-system/tokens). Keeping themes as typed objects makes white-labeling
// straightforward: brand.ts overrides a subset of default.ts.
export interface ThemeTokens {
  name: string;
  colors: {
    primary: string;
    background: string;
    surface: string;
    border: string;
    success: string;
    warning: string;
    destructive: string;
    info: string;
  };
  radius: {
    sm: string;
    md: string;
    lg: string;
  };
  font: {
    sans: string;
    mono: string;
  };
}

export const defaultTheme: ThemeTokens = {
  name: "default",
  colors: {
    primary: "#4F66F0",
    background: "#FFFFFF",
    surface: "#F8F9FC",
    border: "#E2E4EC",
    success: "#0D9464",
    warning: "#F59E0B",
    destructive: "#DC3035",
    info: "#06B6D4",
  },
  radius: {
    sm: "0.5rem",
    md: "0.75rem",
    lg: "1rem",
  },
  font: {
    sans: "var(--font-bakh), var(--font-estedad), ui-sans-serif, system-ui, sans-serif",
    mono: "ui-monospace, SFMono-Regular, Menlo, monospace",
  },
};
