import { redirect } from "@/i18n/navigation";

export default async function LocationsRedirect({
  params,
}: {
  params: Promise<{ locale: string; workspaceSlug: string }>;
}) {
  const { locale } = await params;
  redirect({ href: "/probes", locale });
}
