import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type SurfaceProps = { children: ReactNode; className?: string };

// Keep existing call sites while making all content visible without JavaScript.
export function Reveal({ children, className }: SurfaceProps & { delay?: number }) {
  return <div className={cn("content-group", className)}>{children}</div>;
}
export function SpotlightCard({ children, className }: SurfaceProps) {
  return <div className={cn("surface-wrapper", className)}>{children}</div>;
}
export function HoloCard({ children, className }: SurfaceProps) {
  return <div className={cn("surface-wrapper", className)}>{children}</div>;
}
export function GlassSurface({ children, className }: SurfaceProps) {
  return <div className={cn("plain-surface", className)}>{children}</div>;
}
