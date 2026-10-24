import { redirect } from "next/navigation";
import { getCloudSiteUrl } from "@/lib/cloud-site-url";

export const dynamic = "force-dynamic";

export default function CloudAdminRedirectPage() {
  redirect(`${getCloudSiteUrl()}/admin`);
}
