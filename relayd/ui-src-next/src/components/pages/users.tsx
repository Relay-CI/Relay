"use client";

import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import { getUsers, createUser, updateUser, deleteUser } from "@/lib/api";
import type { User } from "@/lib/api";

type CurrentUser = { username: string; role: string } | null;

interface UsersPageProps {
  currentUser: CurrentUser;
}

export function UsersPage({ currentUser }: UsersPageProps) {
  const isOwner = currentUser?.role === "owner";
  const [users, setUsers] = useState<User[]>([]);
  const [form, setForm] = useState({ username: "", password: "", role: "deployer" });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "danger"; text: string } | null>(null);

  function load() {
    if (!isOwner) return;
    getUsers().then((us) => setUsers(us ?? [])).catch(() => {});
  }

  useEffect(load, [isOwner]);

  if (!isOwner) {
    return <div className="flex items-center justify-center h-full text-white/30 text-sm">Owner access required</div>;
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true); setNotice(null);
    try {
      await createUser({ username: form.username, password: form.password, role: form.role });
      setForm({ username: "", password: "", role: "deployer" });
      setNotice({ tone: "ok", text: "User created." });
      load();
    } catch (err) {
      setNotice({ tone: "danger", text: err instanceof Error ? err.message : "Failed to create user" });
    } finally { setBusy(false); }
  }

  async function handleChangeRole(id: string, role: string) {
    await updateUser(id, { role });
    load();
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this user?")) return;
    await deleteUser(id);
    load();
  }

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Team</div>
        <h1 className="text-xl font-semibold text-white">User management</h1>
      </div>

      {/* User list */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-white/[0.06]">
          <div className="text-base font-semibold text-white">Accounts</div>
        </div>
        {!users.length ? (
          <div className="text-sm text-white/30 text-center py-8">No users found.</div>
        ) : (
          <div className="divide-y divide-white/[0.04]">
            {users.map((u) => (
              <div key={u.id} className="flex items-center gap-4 px-5 py-3">
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-white">{u.username}</div>
                  <div className="text-xs text-white/35">{u.role}</div>
                </div>
                <select
                  value={u.role}
                  onChange={(e) => handleChangeRole(u.id, e.target.value)}
                  className="text-xs bg-white/[0.04] border border-white/[0.08] rounded px-2 py-1 text-white/70 focus:outline-none"
                >
                  <option value="owner">owner</option>
                  <option value="deployer">deployer</option>
                  <option value="viewer">viewer</option>
                </select>
                <button
                  type="button"
                  onClick={() => handleDelete(u.id)}
                  disabled={u.username === currentUser?.username}
                  className="text-xs text-white/40 hover:text-red-400 transition-colors px-2 py-1 rounded hover:bg-white/[0.04] disabled:opacity-30 disabled:cursor-not-allowed"
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create user form */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div className="eyebrow mb-1">Add user</div>
        <form onSubmit={handleCreate} className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <Field label="Username">
              <input
                className="text-input"
                value={form.username}
                onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
                required
                autoComplete="off"
              />
            </Field>
            <Field label="Password">
              <input
                type="password"
                className="text-input"
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                required
                minLength={8}
                autoComplete="new-password"
              />
            </Field>
            <Field label="Role">
              <select
                className="text-input"
                value={form.role}
                onChange={(e) => setForm((f) => ({ ...f, role: e.target.value }))}
              >
                <option value="owner">owner</option>
                <option value="deployer">deployer</option>
                <option value="viewer">viewer</option>
              </select>
            </Field>
          </div>

          {notice && (
            <div className={cn("rounded px-3 py-2 text-sm border", notice.tone === "ok" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : "bg-red-500/10 border-red-500/30 text-red-400")}>
              {notice.text}
            </div>
          )}

          <button type="submit" disabled={busy} className="text-sm bg-white text-black font-semibold px-4 py-2 rounded hover:bg-white/90 transition-colors disabled:opacity-40">
            {busy ? "Creating…" : "Create user"}
          </button>
        </form>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="text-xs text-white/40 mb-1.5">{label}</div>
      {children}
    </label>
  );
}
