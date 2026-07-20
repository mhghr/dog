import { redirect } from "@/i18n/navigation";

export default async function MonitorsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  redirect({ href: "/app/nodes", locale });
}
