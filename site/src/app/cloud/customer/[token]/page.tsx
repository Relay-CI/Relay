import { redirect } from "next/navigation";
import { getCloudSiteUrl } from "@/lib/cloud-site-url";

export const dynamic = "force-dynamic";

type CustomerRedirectPageProps = {
  params: Promise<{ token: string }>;
};

export default async function CustomerRedirectPage({
  params,
}: CustomerRedirectPageProps) {
  const { token } = await params;
  redirect(`${getCloudSiteUrl()}/customer/${token}`);
}
