import type { MetadataRoute } from "next";
import { siteEnv } from "@/lib/env";

export const dynamic = "force-dynamic";

const routes = [
  "",
  "/about",
  "/architecture",
  "/cloud",
  "/contact",
  "/docs",
  "/docs/agent",
  "/docs/architecture",
  "/docs/cli",
  "/docs/github",
  "/docs/variants",
  "/docs/scaling",
  "/docs/operations",
  "/docs/mcp",
  "/docs/plugins",
  "/docs/previews",
  "/docs/relaydb",
  "/docs/services",
  "/features",
  "/get-started",
  "/github",
  "/plugins",
  "/previews",
  "/relay",
  "/relaydb",
  "/relayd",
  "/services",
];

export default function sitemap(): MetadataRoute.Sitemap {
  return routes.map((route) => ({
    url: `${siteEnv.siteUrl}${route || "/"}`,
    lastModified: new Date(),
    changeFrequency: route === "" ? "weekly" : "monthly",
    priority: route === "" ? 1 : route.startsWith("/docs") ? 0.8 : 0.7,
  }));
}
