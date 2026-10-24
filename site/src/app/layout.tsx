import type { Metadata } from "next";
import "./globals.css";
import "./quiet.css";
import "@fontsource-variable/inter";
import "./graphite.css";
import { TooltipProvider } from "@/components/ui/tooltip";

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL || "http://localhost:3000"),
  title: {
    default: "Relay",
    template: "%s | Relay",
  },
  applicationName: "Relay",
  description:
    "Self-hosted deployment lanes with protected dev and staging environments, streamed logs, rollback-ready releases, and direct control over your own servers.",
  keywords: [
    "relay",
    "self-hosted deploys",
    "deployment lanes",
    "preview lanes",
    "dev staging production",
    "blue-green deploys",
    "edge access policies",
    "build logs",
    "docker deploys",
    "station runtime",
  ],
  category: "technology",
  alternates: {
    canonical: "/",
  },
  openGraph: {
    type: "website",
    url: "/",
    title: "Relay",
    description:
      "Self-hosted deployment lanes with protected dev and staging environments, streamed logs, rollback-ready releases, and direct control over your own servers.",
    siteName: "Relay",
  },
  twitter: {
    card: "summary_large_image",
    title: "Relay",
    description:
      "Self-hosted deployment lanes with protected dev and staging environments, streamed logs, rollback-ready releases, and direct control over your own servers.",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-image-preview": "large",
      "max-snippet": -1,
      "max-video-preview": -1,
    },
  },
  icons: {
    icon: [
      { url: "/relay-app-icon.png", type: "image/png" },
      { url: "/relay-icon.svg", type: "image/svg+xml" },
    ],
    shortcut: "/relay-app-icon.png",
    apple: "/relay-app-icon.png",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="font-sans">
      <body className="antialiased">
        <TooltipProvider>{children}</TooltipProvider>
      </body>
    </html>
  );
}
