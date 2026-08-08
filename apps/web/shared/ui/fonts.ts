import localFont from "next/font/local";

export const inter = localFont({
  src: [{ path: "../../fonts/Inter-Variable.woff2", weight: "100 900", style: "normal" }],
  variable: "--font-inter",
  display: "swap",
});

export const estedad = localFont({
  src: [
    { path: "../../fonts/Estedad-Light.woff2", weight: "300", style: "normal" },
    { path: "../../fonts/Estedad-Regular.woff2", weight: "400", style: "normal" },
    { path: "../../fonts/Estedad-Medium.woff2", weight: "500", style: "normal" },
    { path: "../../fonts/Estedad-SemiBold.woff2", weight: "600", style: "normal" },
    { path: "../../fonts/Estedad-Bold.woff2", weight: "700", style: "normal" },
  ],
  variable: "--font-estedad",
  display: "swap",
});

export const bakh = localFont({
  src: [
    { path: "../../fonts/bakh-light.woff2", weight: "300", style: "normal" },
    { path: "../../fonts/bakh.woff2", weight: "400", style: "normal" },
    { path: "../../fonts/bakh-medium.woff2", weight: "500", style: "normal" },
    { path: "../../fonts/bakh-semibold.woff2", weight: "600", style: "normal" },
    { path: "../../fonts/bakh-bold.woff2", weight: "700", style: "normal" },
  ],
  variable: "--font-bakh",
  display: "swap",
  // Bakh ships unbalanced vertical metrics (ascent 100% / descent 55%),
  // which pushes text off-center inside flex-centered controls. Override
  // to Estedad's measured ratios so line boxes center like before.
  declarations: [
    { prop: "ascent-override", value: "117%" },
    { prop: "descent-override", value: "59%" },
    { prop: "line-gap-override", value: "0%" },
  ],
});

// Single-weight display face; the full weight range prevents faux-bold
// synthesis when headings use bold utility classes.
export const plasma = localFont({
  src: [{ path: "../../fonts/plasma.woff2", weight: "100 900", style: "normal" }],
  variable: "--font-plasma",
  display: "swap",
});
