import { setRequestLocale } from "next-intl/server";

import HeroSection from "@/components/marketing/hero-section";
import MonitorTypesSection from "@/components/marketing/monitor-types-section";
import FeaturesSection from "@/components/marketing/features-section";
import HowItWorksSection from "@/components/marketing/how-it-works-section";
import CTASection from "@/components/marketing/cta-section";

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
