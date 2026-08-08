import { redirect } from "@/i18n/navigation";

export default async function ConsoleIndexPage({
  params,
}: {
  params: Promise<{ locale: string; workspaceSlug: string }>;
}) {
  const { locale, workspaceSlug } = await params;
  redirect({ href: `/console/w/${workspaceSlug}/dashboard`, locale });
}
