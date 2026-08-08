import { setRequestLocale } from "next-intl/server";

import HeroSection from "@/features/marketing/hero-section";
import MonitorTypesSection from "@/features/marketing/monitor-types-section";
import FeaturesSection from "@/features/marketing/features-section";
import HowItWorksSection from "@/features/marketing/how-it-works-section";
import CTASection from "@/features/marketing/cta-section";

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
