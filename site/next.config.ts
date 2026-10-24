import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactCompiler: true,
  output: "standalone",
  experimental: {
    optimizePackageImports: ["lucide-react"],
  },
  async redirects() {
    return [
      { source: "/relay",        destination: "/docs/cli",          permanent: true },
      { source: "/relayd",       destination: "/docs/agent",        permanent: true },
      { source: "/architecture", destination: "/docs/architecture", permanent: true },
      { source: "/plugins",      destination: "/docs/plugins",      permanent: true },
      { source: "/github",       destination: "/docs/github",       permanent: true },
      { source: "/services",     destination: "/docs/services",     permanent: true },
      { source: "/previews",     destination: "/docs/previews",     permanent: true },
    ];
  },
};

export default nextConfig;
