import { SVGProps } from "react";

interface RelayMarkProps extends SVGProps<SVGSVGElement> {
  title?: string;
}

export function RelayMark({ title, ...props }: RelayMarkProps) {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden={!title}
      role={title ? "img" : undefined}
      {...props}
    >
      {title && <title>{title}</title>}
      {/* R cap and bowl */}
      <path
        d="M10 8H21Q26 8 26 12Q26 19 21 19L17 20"
        stroke="currentColor"
        strokeWidth="3.5"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* R leg */}
      <path
        d="M17 20L23 28"
        stroke="currentColor"
        strokeWidth="3.5"
        fill="none"
        strokeLinecap="round"
      />
      {/* Crescent arc through hub */}
      <path
        d="M10 8Q5 10 7 15Q9 20 17 20"
        stroke="currentColor"
        strokeWidth="3"
        fill="none"
        strokeLinecap="round"
      />
      {/* Hub node */}
      <circle cx="7" cy="15" r="3" fill="currentColor" />
      {/* Terminal node */}
      <circle cx="23" cy="28" r="2.5" fill="currentColor" />
    </svg>
  );
}
