import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactCompiler: true,
  output: "export",
  trailingSlash: true,
  basePath: "/dashboard",
  assetPrefix: "/dashboard",
  images: {
    unoptimized: true,
  },
};

export default nextConfig;
