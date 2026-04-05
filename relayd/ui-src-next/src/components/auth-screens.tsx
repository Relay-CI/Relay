"use client";

import { useEffect, useState } from "react";
import { RelayMark } from "@/components/relay-mark";
import { login, setup } from "@/lib/api";
import { cn } from "@/lib/utils";

interface AuthShellProps {
  children: React.ReactNode;
  signal: string;
  signalVariant?: "teal" | "amber" | "success";
}

function AuthShell({ children, signal, signalVariant = "teal" }: AuthShellProps) {
  return (
    <div className="min-h-screen bg-black flex items-center justify-center p-4 relative overflow-hidden">
      <div
        className="absolute inset-0 pointer-events-none"
        style={{
          backgroundImage: "linear-gradient(rgba(220,60,60,0.06) 1px, transparent 1px), linear-gradient(90deg, rgba(220,60,60,0.06) 1px, transparent 1px)",
          backgroundSize: "40px 40px",
        }}
      />
      <div
        className="absolute inset-0 pointer-events-none"
        style={{ background: "radial-gradient(ellipse at center, transparent 40%, rgba(0,0,0,0.8) 100%)" }}
      />

      <div className="relative z-10 w-full max-w-sm">
        <div className="flex flex-col items-center mb-8 gap-3">
          <RelayMark className="w-10 h-10 text-white" />
          <div
            className={cn("text-[10px] font-semibold uppercase tracking-widest px-3 py-1 rounded-full border", {
              "border-relay-teal/30 text-relay-teal bg-relay-teal/10": signalVariant === "teal",
              "border-amber-500/30 text-amber-400 bg-amber-500/10": signalVariant === "amber",
              "border-emerald-500/30 text-emerald-400 bg-emerald-500/10": signalVariant === "success",
            })}
          >
            {signal}
          </div>
        </div>
        {children}
      </div>
    </div>
  );
}

interface LoginPageProps {
  legacyMode?: boolean;
  showSetupOption?: boolean;
  onShowSetup?: () => void;
}

export function LoginPage({ legacyMode = false, showSetupOption = false, onShowSetup }: LoginPageProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const [cliMode, setCliMode] = useState(false);
  const [cliPort, setCliPort] = useState<number | null>(null);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const parsedPort = Number.parseInt(params.get("port") ?? "", 10);
    const validPort = Number.isInteger(parsedPort) && parsedPort > 0 && parsedPort <= 65535;
    setCliMode(params.get("cli") === "1" && validPort);
    setCliPort(validPort ? parsedPort : null);
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setPending(true);
    setError("");
    try {
      const response = legacyMode
        ? await login(token)
        : await login({ username, password, cliPort: cliPort ?? undefined });

      if (cliMode && !legacyMode) {
        if (response.cli_redirect) {
          window.location.assign(response.cli_redirect);
          return;
        }
        setError("CLI login handshake failed.");
        return;
      }

      window.location.reload();
    } catch (err) {
      setError((err as Error).message || "Sign in failed.");
    } finally {
      setPending(false);
    }
  }

  return (
    <AuthShell signal="Secure dashboard access" signalVariant="teal">
      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="bg-zinc-950 border border-white/[0.08] rounded-xl p-6">
          <div className="mb-5">
            <h1 className="text-lg font-semibold text-white">Sign in to Relayd</h1>
            <p className="text-sm text-white/50 mt-1">
              {legacyMode
                ? "Enter your relay token to open the admin panel."
                : cliMode
                  ? "Use your operator account. After sign-in, the browser will return control to Relay CLI."
                  : "Use your operator account to manage environments and deployments."}
            </p>
          </div>

          <div className="space-y-3">
            {legacyMode ? (
              <input
                type="password"
                autoComplete="current-password"
                placeholder="Paste relay token"
                className="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50 transition-colors"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                required
              />
            ) : (
              <>
                <input
                  type="text"
                  autoComplete="username"
                  placeholder="Username"
                  className="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50 transition-colors"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                />
                <input
                  type="password"
                  autoComplete="current-password"
                  placeholder="Password"
                  className="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50 transition-colors"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </>
            )}

            {error && (
              <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={pending || (legacyMode ? !token : !username || !password)}
              className="w-full bg-white text-black font-semibold text-sm py-2.5 rounded-lg hover:bg-white/90 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
            >
              {pending ? "Signing in..." : cliMode && !legacyMode ? "Sign in and Return to CLI" : "Sign in"}
            </button>

            {legacyMode && (
              <div className="text-xs text-white/30 text-center">
                Token source: <span className="font-mono bg-white/[0.06] px-1.5 py-0.5 rounded">data/token.txt</span>
              </div>
            )}

            {showSetupOption && onShowSetup && (
              <button
                type="button"
                onClick={onShowSetup}
                className="w-full text-xs text-white/45 hover:text-white transition-colors"
              >
                Create the first owner account instead
              </button>
            )}
          </div>
        </div>
      </form>
    </AuthShell>
  );
}

interface SetupPageProps {
  onShowLogin?: () => void;
}

export function SetupPage({ onShowLogin }: SetupPageProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const mismatch = confirm !== "" && password !== confirm;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (mismatch) return;
    setPending(true);
    setError("");
    try {
      await setup({ username, password });
      window.location.reload();
    } catch (err) {
      setError((err as Error).message || "Setup failed.");
    } finally {
      setPending(false);
    }
  }

  return (
    <AuthShell signal="Owner account setup" signalVariant="success">
      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="bg-zinc-950 border border-white/[0.08] rounded-xl p-6">
          <div className="mb-5">
            <div className="text-[10px] uppercase tracking-widest text-emerald-500 font-semibold mb-2">First-time setup</div>
            <h1 className="text-lg font-semibold text-white">Create the owner account</h1>
            <p className="text-sm text-white/50 mt-1">
              No accounts exist yet. Set up the initial owner account before the dashboard starts accepting logins.
            </p>
          </div>

          <div className="space-y-3">
            <input
              type="text"
              autoComplete="username"
              placeholder="Username"
              className="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50 transition-colors"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
            <input
              type="password"
              autoComplete="new-password"
              placeholder="Password (min 8 chars)"
              className="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-white placeholder:text-white/30 outline-none focus:border-relay-accent/50 transition-colors"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
            <input
              type="password"
              autoComplete="new-password"
              placeholder="Confirm password"
              className={cn(
                "w-full bg-white/[0.04] border rounded-lg px-3 py-2.5 text-sm text-white placeholder:text-white/30 outline-none transition-colors",
                mismatch ? "border-red-500/50" : "border-white/[0.08] focus:border-relay-accent/50",
              )}
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
            />

            {mismatch && <div className="text-sm text-red-400">Passwords do not match.</div>}
            {error && (
              <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={!username || password.length < 8 || mismatch || pending}
              className="w-full bg-white text-black font-semibold text-sm py-2.5 rounded-lg hover:bg-white/90 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
            >
              {pending ? "Creating account..." : "Create account"}
            </button>

            {onShowLogin && (
              <button
                type="button"
                onClick={onShowLogin}
                className="w-full text-xs text-white/45 hover:text-white transition-colors"
              >
                Use the relay token login instead
              </button>
            )}
          </div>
        </div>
      </form>
    </AuthShell>
  );
}
