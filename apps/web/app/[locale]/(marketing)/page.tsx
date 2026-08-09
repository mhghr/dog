import { setRequestLocale } from "next-intl/server";
import dynamic from "next/dynamic";

import HeroSection from "@/features/marketing/hero-section";

const MonitorTypesSection = dynamic(() => import("@/features/marketing/monitor-types-section"), { ssr: true });
const FeaturesSection = dynamic(() => import("@/features/marketing/features-section"), { ssr: true });
const HowItWorksSection = dynamic(() => import("@/features/marketing/how-it-works-section"), { ssr: true });
const CTASection = dynamic(() => import("@/features/marketing/cta-section"), { ssr: true });

export default async function LandingPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);

  return (
    <div className="overflow-x-clip">
      <HeroSection />
      <MonitorTypesSection />
      <FeaturesSection />
      <HowItWorksSection />
      <CTASection />
    </div>
  );
}
