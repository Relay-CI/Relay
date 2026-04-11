"use client";

import { createContext, useContext, useEffect, useRef, useState, useCallback } from "react";
import { getPublicTheme } from "@/lib/api";
import { buildThemeCSS, BUILT_IN_THEMES } from "@/lib/themes";

interface ThemeContextValue {
  themeName: string;
  customCSS: string;
  /** Preview a theme live without persisting. Pass "" to revert to server state. */
  previewTheme: (name: string, css?: string) => void;
  /** Clear preview and revert to last saved server theme. */
  clearPreview: () => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  themeName: "default",
  customCSS: "",
  previewTheme: () => {},
  clearPreview: () => {},
});

export function useTheme() {
  return useContext(ThemeContext);
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [serverThemeName, setServerThemeName] = useState("default");
  const [serverCustomCSS, setServerCustomCSS] = useState("");
  const [previewName, setPreviewName] = useState<string | null>(null);
  const [previewCSS, setPreviewCSS] = useState<string | null>(null);
  const styleRef = useRef<HTMLStyleElement | null>(null);

  // Fetch theme from server once on mount
  useEffect(() => {
    getPublicTheme()
      .then((data) => {
        setServerThemeName(data.theme_name ?? "default");
        setServerCustomCSS(data.theme_css ?? "");
      })
      .catch(() => {
        // Server unreachable or unauthenticated — use defaults
      });
  }, []);

  // Inject/update the <style> tag whenever active theme changes
  const activeName = previewName ?? serverThemeName;
  const activeCSS = previewCSS ?? serverCustomCSS;

  useEffect(() => {
    const css = buildThemeCSS(activeName, activeCSS);
    if (!styleRef.current) {
      const el = document.createElement("style");
      el.id = "relay-theme";
      document.head.appendChild(el);
      styleRef.current = el;
    }
    styleRef.current.textContent = css;
  }, [activeName, activeCSS]);

  const previewTheme = useCallback((name: string, css = "") => {
    setPreviewName(name || null);
    setPreviewCSS(css || null);
  }, []);

  const clearPreview = useCallback(() => {
    setPreviewName(null);
    setPreviewCSS(null);
  }, []);

  return (
    <ThemeContext.Provider
      value={{
        themeName: serverThemeName,
        customCSS: serverCustomCSS,
        previewTheme,
        clearPreview,
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
}

/**
 * Refresh the ThemeProvider's server state after saving new theme settings.
 * Called from server-settings after a successful save so the theme
 * updates immediately without a full page reload.
 */
export function refreshThemeFromServer(
  name: string,
  css: string,
  ctx: ThemeContextValue,
) {
  // Clear any active preview so the saved theme takes effect
  ctx.clearPreview();
  // The provider will re-read from server on next mount; for immediate update
  // we also apply the preset+custom CSS directly via previewTheme, then mark it
  // as the "saved" state by clearing preview after a tick so the style stays.
  //
  // Simpler: just apply it as a preview — the user will see it immediately.
  // The next page load will re-fetch from server and apply the same CSS.
  ctx.previewTheme(name, css);
}

// Re-export so consumers don't need to import themes.ts separately
export { BUILT_IN_THEMES };
