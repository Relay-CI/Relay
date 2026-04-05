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
      {/* Relay mark: stylized R with a relay signal arc */}
      <rect x="4" y="4" width="10" height="24" rx="1.5" fill="currentColor" />
      <path
        d="M14 4 Q28 4 28 11 Q28 18 14 18"
        stroke="currentColor"
        strokeWidth="3"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M14 18 L24 28"
        stroke="currentColor"
        strokeWidth="3"
        strokeLinecap="round"
      />
      {/* Signal arc */}
      <path
        d="M20 8 Q32 16 20 24"
        stroke="var(--relay-accent, #cc2222)"
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
        opacity="0.7"
      />
    </svg>
  );
}
