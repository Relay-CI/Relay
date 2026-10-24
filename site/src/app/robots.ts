import type { MetadataRoute } from "next";
import { siteEnv } from "@/lib/env";

export const dynamic = "force-dynamic";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
    },
    sitemap: `${siteEnv.siteUrl}/sitemap.xml`,
    host: siteEnv.siteUrl,
  };
}
