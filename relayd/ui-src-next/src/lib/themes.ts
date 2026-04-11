// Built-in theme presets for the Relay control room.
// Each theme is a CSS override block injected into :root.
// Custom themes (raw CSS) can also be written by the user in server settings.

export interface ThemePreset {
  id: string;
  name: string;
  description: string;
  swatches: string[]; // preview colors [bg, accent, text]
  css: string;
}

export const BUILT_IN_THEMES: ThemePreset[] = [
  {
    id: "default",
    name: "Relay Dark",
    description: "Default red-on-black",
    swatches: ["#000000", "#cc2222", "#f0eaea"],
    css: "", // no overrides — uses base globals.css
  },
  {
    id: "midnight-blue",
    name: "Midnight Blue",
    description: "Blue accent on deep navy",
    swatches: ["#020408", "#2563eb", "#e8edf8"],
    css: `
:root {
  --relay-bg:               #020408;
  --relay-panel:            rgba(4, 8, 18, 0.97);
  --relay-panel-strong:     rgba(5, 10, 22, 0.99);
  --relay-line:             rgba(59, 130, 246, 0.15);
  --relay-line-strong:      rgba(59, 130, 246, 0.28);
  --relay-text:             #e8edf8;
  --relay-muted:            #6880a0;
  --relay-muted-deep:       #2c3a50;
  --relay-accent:           #2563eb;
  --relay-accent-bright:    #3b82f6;
  --relay-teal:             #06b6d4;
  --relay-teal-soft:        #67e8f9;
  --relay-success:          #34d399;
  --relay-warn:             #fb923c;
  --relay-danger:           #ef4444;
  --background:             oklch(0.05 0.012 240);
  --foreground:             oklch(0.93 0.008 220);
  --card:                   oklch(0.08 0.014 240);
  --card-foreground:        oklch(0.93 0.008 220);
  --popover:                oklch(0.07 0.013 240);
  --popover-foreground:     oklch(0.93 0.008 220);
  --primary:                oklch(0.52 0.22 255);
  --primary-foreground:     oklch(0.98 0 0);
  --secondary:              oklch(0.12 0.015 240);
  --secondary-foreground:   oklch(0.93 0.008 220);
  --muted:                  oklch(0.12 0.015 240);
  --muted-foreground:       oklch(0.58 0.025 240);
  --accent:                 oklch(0.14 0.016 240);
  --accent-foreground:      oklch(0.93 0.008 220);
  --destructive:            oklch(0.52 0.22 30);
  --destructive-foreground: oklch(0.98 0 0);
  --border:                 oklch(0.20 0.025 250 / 60%);
  --input:                  oklch(0.11 0.015 240);
  --ring:                   oklch(0.52 0.22 255);
  --sidebar:                oklch(0.06 0.012 240);
  --sidebar-foreground:     oklch(0.93 0.008 220);
  --sidebar-primary:        oklch(0.52 0.22 255);
  --sidebar-primary-foreground: oklch(0.98 0 0);
  --sidebar-accent:         oklch(0.10 0.014 240);
  --sidebar-accent-foreground: oklch(0.93 0.008 220);
  --sidebar-border:         oklch(0.18 0.022 250 / 40%);
  --sidebar-ring:           oklch(0.52 0.22 255);
}`,
  },
  {
    id: "forest",
    name: "Forest",
    description: "Green accent on dark slate",
    swatches: ["#030806", "#16a34a", "#e2f0e8"],
    css: `
:root {
  --relay-bg:               #030806;
  --relay-panel:            rgba(4, 12, 8, 0.97);
  --relay-panel-strong:     rgba(5, 15, 10, 0.99);
  --relay-line:             rgba(34, 197, 94, 0.13);
  --relay-line-strong:      rgba(34, 197, 94, 0.24);
  --relay-text:             #e2f0e8;
  --relay-muted:            #5a8070;
  --relay-muted-deep:       #1e3828;
  --relay-accent:           #16a34a;
  --relay-accent-bright:    #22c55e;
  --relay-teal:             #0891b2;
  --relay-teal-soft:        #22d3ee;
  --relay-success:          #4ade80;
  --relay-warn:             #facc15;
  --relay-danger:           #f87171;
  --background:             oklch(0.05 0.012 150);
  --foreground:             oklch(0.93 0.012 140);
  --card:                   oklch(0.08 0.014 150);
  --card-foreground:        oklch(0.93 0.012 140);
  --popover:                oklch(0.07 0.013 150);
  --popover-foreground:     oklch(0.93 0.012 140);
  --primary:                oklch(0.52 0.18 145);
  --primary-foreground:     oklch(0.98 0 0);
  --secondary:              oklch(0.12 0.015 150);
  --secondary-foreground:   oklch(0.93 0.012 140);
  --muted:                  oklch(0.12 0.015 150);
  --muted-foreground:       oklch(0.58 0.02 145);
  --accent:                 oklch(0.14 0.016 150);
  --accent-foreground:      oklch(0.93 0.012 140);
  --destructive:            oklch(0.58 0.18 30);
  --destructive-foreground: oklch(0.98 0 0);
  --border:                 oklch(0.20 0.022 150 / 60%);
  --input:                  oklch(0.11 0.014 150);
  --ring:                   oklch(0.52 0.18 145);
  --sidebar:                oklch(0.06 0.012 150);
  --sidebar-foreground:     oklch(0.93 0.012 140);
  --sidebar-primary:        oklch(0.52 0.18 145);
  --sidebar-primary-foreground: oklch(0.98 0 0);
  --sidebar-accent:         oklch(0.10 0.014 150);
  --sidebar-accent-foreground: oklch(0.93 0.012 140);
  --sidebar-border:         oklch(0.18 0.020 150 / 40%);
  --sidebar-ring:           oklch(0.52 0.18 145);
}`,
  },
  {
    id: "amber",
    name: "Amber",
    description: "Warm amber on charcoal",
    swatches: ["#0a0804", "#d97706", "#f5f0e8"],
    css: `
:root {
  --relay-bg:               #0a0804;
  --relay-panel:            rgba(16, 12, 6, 0.97);
  --relay-panel-strong:     rgba(20, 14, 7, 0.99);
  --relay-line:             rgba(251, 191, 36, 0.14);
  --relay-line-strong:      rgba(251, 191, 36, 0.26);
  --relay-text:             #f5f0e8;
  --relay-muted:            #9a8060;
  --relay-muted-deep:       #4a3820;
  --relay-accent:           #d97706;
  --relay-accent-bright:    #f59e0b;
  --relay-teal:             #0891b2;
  --relay-teal-soft:        #38bdf8;
  --relay-success:          #65a30d;
  --relay-warn:             #ea580c;
  --relay-danger:           #dc2626;
  --background:             oklch(0.07 0.015 60);
  --foreground:             oklch(0.94 0.012 70);
  --card:                   oklch(0.10 0.016 60);
  --card-foreground:        oklch(0.94 0.012 70);
  --popover:                oklch(0.09 0.015 60);
  --popover-foreground:     oklch(0.94 0.012 70);
  --primary:                oklch(0.60 0.18 65);
  --primary-foreground:     oklch(0.10 0 0);
  --secondary:              oklch(0.14 0.016 60);
  --secondary-foreground:   oklch(0.94 0.012 70);
  --muted:                  oklch(0.14 0.016 60);
  --muted-foreground:       oklch(0.60 0.025 65);
  --accent:                 oklch(0.16 0.018 60);
  --accent-foreground:      oklch(0.94 0.012 70);
  --destructive:            oklch(0.55 0.20 30);
  --destructive-foreground: oklch(0.98 0 0);
  --border:                 oklch(0.22 0.022 65 / 60%);
  --input:                  oklch(0.12 0.015 60);
  --ring:                   oklch(0.60 0.18 65);
  --sidebar:                oklch(0.08 0.014 60);
  --sidebar-foreground:     oklch(0.94 0.012 70);
  --sidebar-primary:        oklch(0.60 0.18 65);
  --sidebar-primary-foreground: oklch(0.10 0 0);
  --sidebar-accent:         oklch(0.12 0.016 60);
  --sidebar-accent-foreground: oklch(0.94 0.012 70);
  --sidebar-border:         oklch(0.20 0.020 65 / 40%);
  --sidebar-ring:           oklch(0.60 0.18 65);
}`,
  },
  {
    id: "synthwave",
    name: "Synthwave",
    description: "Purple/pink neon on near-black",
    swatches: ["#03010a", "#9333ea", "#f0e8ff"],
    css: `
:root {
  --relay-bg:               #03010a;
  --relay-panel:            rgba(8, 4, 20, 0.97);
  --relay-panel-strong:     rgba(10, 5, 25, 0.99);
  --relay-line:             rgba(168, 85, 247, 0.15);
  --relay-line-strong:      rgba(168, 85, 247, 0.28);
  --relay-text:             #f0e8ff;
  --relay-muted:            #8060a8;
  --relay-muted-deep:       #32204a;
  --relay-accent:           #9333ea;
  --relay-accent-bright:    #a855f7;
  --relay-teal:             #ec4899;
  --relay-teal-soft:        #f472b6;
  --relay-success:          #34d399;
  --relay-warn:             #fbbf24;
  --relay-danger:           #f43f5e;
  --background:             oklch(0.05 0.018 290);
  --foreground:             oklch(0.93 0.015 280);
  --card:                   oklch(0.08 0.020 290);
  --card-foreground:        oklch(0.93 0.015 280);
  --popover:                oklch(0.07 0.018 290);
  --popover-foreground:     oklch(0.93 0.015 280);
  --primary:                oklch(0.55 0.26 295);
  --primary-foreground:     oklch(0.98 0 0);
  --secondary:              oklch(0.12 0.020 290);
  --secondary-foreground:   oklch(0.93 0.015 280);
  --muted:                  oklch(0.12 0.020 290);
  --muted-foreground:       oklch(0.58 0.030 290);
  --accent:                 oklch(0.14 0.022 290);
  --accent-foreground:      oklch(0.93 0.015 280);
  --destructive:            oklch(0.55 0.24 10);
  --destructive-foreground: oklch(0.98 0 0);
  --border:                 oklch(0.20 0.030 290 / 60%);
  --input:                  oklch(0.11 0.018 290);
  --ring:                   oklch(0.55 0.26 295);
  --sidebar:                oklch(0.06 0.018 290);
  --sidebar-foreground:     oklch(0.93 0.015 280);
  --sidebar-primary:        oklch(0.55 0.26 295);
  --sidebar-primary-foreground: oklch(0.98 0 0);
  --sidebar-accent:         oklch(0.10 0.020 290);
  --sidebar-accent-foreground: oklch(0.93 0.015 280);
  --sidebar-border:         oklch(0.18 0.028 290 / 40%);
  --sidebar-ring:           oklch(0.55 0.26 295);
}`,
  },
  {
    id: "light",
    name: "Light",
    description: "Clean light mode",
    swatches: ["#f8f8f8", "#1d4ed8", "#111111"],
    css: `
:root {
  --relay-bg:               #f8f8f8;
  --relay-panel:            rgba(255, 255, 255, 0.97);
  --relay-panel-strong:     rgba(248, 248, 248, 0.99);
  --relay-line:             rgba(0, 0, 0, 0.08);
  --relay-line-strong:      rgba(0, 0, 0, 0.15);
  --relay-text:             #111111;
  --relay-muted:            #666666;
  --relay-muted-deep:       #aaaaaa;
  --relay-accent:           #1d4ed8;
  --relay-accent-bright:    #2563eb;
  --relay-teal:             #0891b2;
  --relay-teal-soft:        #06b6d4;
  --relay-success:          #15803d;
  --relay-warn:             #b45309;
  --relay-danger:           #dc2626;
  --background:             oklch(0.98 0 0);
  --foreground:             oklch(0.12 0 0);
  --card:                   oklch(1 0 0);
  --card-foreground:        oklch(0.12 0 0);
  --popover:                oklch(1 0 0);
  --popover-foreground:     oklch(0.12 0 0);
  --primary:                oklch(0.46 0.20 255);
  --primary-foreground:     oklch(0.98 0 0);
  --secondary:              oklch(0.94 0 0);
  --secondary-foreground:   oklch(0.12 0 0);
  --muted:                  oklch(0.94 0 0);
  --muted-foreground:       oklch(0.50 0 0);
  --accent:                 oklch(0.94 0 0);
  --accent-foreground:      oklch(0.12 0 0);
  --destructive:            oklch(0.55 0.22 30);
  --destructive-foreground: oklch(0.98 0 0);
  --border:                 oklch(0.86 0 0);
  --input:                  oklch(0.94 0 0);
  --ring:                   oklch(0.46 0.20 255);
  --sidebar:                oklch(0.96 0 0);
  --sidebar-foreground:     oklch(0.12 0 0);
  --sidebar-primary:        oklch(0.46 0.20 255);
  --sidebar-primary-foreground: oklch(0.98 0 0);
  --sidebar-accent:         oklch(0.92 0 0);
  --sidebar-accent-foreground: oklch(0.12 0 0);
  --sidebar-border:         oklch(0.88 0 0);
  --sidebar-ring:           oklch(0.46 0.20 255);
}
body { background-color: var(--relay-bg); color: var(--relay-text); }
.text-input { background: rgba(0,0,0,0.04); border-color: rgba(0,0,0,0.12); color: #111; }
.text-input::placeholder { color: rgba(0,0,0,0.3); }
.text-input:focus { border-color: rgba(0,0,0,0.25); }
.ghost-btn { color: rgba(0,0,0,0.55); border-color: rgba(0,0,0,0.14); }
.ghost-btn:hover { color: #000; background: rgba(0,0,0,0.05); border-color: rgba(0,0,0,0.22); }
.seg-control { background: rgba(0,0,0,0.06); border-color: rgba(0,0,0,0.10); }
.seg-btn { color: rgba(0,0,0,0.38); }
.seg-btn:hover { color: rgba(0,0,0,0.70); }
.seg-btn--active { background: #111; color: #fff; }`,
  },
];

export const BUILT_IN_THEME_IDS = BUILT_IN_THEMES.map((t) => t.id);

export function getThemeById(id: string): ThemePreset | undefined {
  return BUILT_IN_THEMES.find((t) => t.id === id);
}

/**
 * Builds the complete CSS to inject into the page for a given theme name
 * combined with any user-supplied custom CSS override.
 */
export function buildThemeCSS(themeName: string, customCSS: string): string {
  const preset = getThemeById(themeName);
  const presetCSS = preset?.css ?? "";
  const parts = [presetCSS.trim(), customCSS.trim()].filter(Boolean);
  return parts.join("\n");
}
