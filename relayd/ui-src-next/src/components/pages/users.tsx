"use client";

import { useCallback, useEffect, useState } from "react";
import { cn } from "@/lib/utils";
import {
  getUsers,
  createUser,
  updateUser,
  deleteUser,
  type User,
  type UserPermission,
} from "@/lib/api";

type CurrentUser = { username: string; role: string } | null;

interface UsersPageProps {
  currentUser: CurrentUser;
}

type UserDraft = {
  role: string;
  permissionsText: string;
  saving?: boolean;
};

export function UsersPage({ currentUser }: UsersPageProps) {
  const isOwner = currentUser?.role === "owner";
  const [users, setUsers] = useState<User[]>([]);
  const [drafts, setDrafts] = useState<Record<string, UserDraft>>({});
  const [form, setForm] = useState({
    username: "",
    password: "",
    role: "deployer",
    permissionsText: "",
  });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "danger"; text: string } | null>(null);

  const load = useCallback(async () => {
    if (!isOwner) return;
    try {
      const nextUsers = await getUsers();
      setUsers(nextUsers ?? []);
      setDrafts(
        Object.fromEntries(
          (nextUsers ?? []).map((user) => [
            user.id,
            {
              role: user.role,
              permissionsText: formatPermissions(user.permissions ?? []),
            },
          ]),
        ),
      );
    } catch {
      // ignore; page already shows current state
    }
  }, [isOwner]);

  useEffect(() => {
    void load();
  }, [load]);

  if (!isOwner) {
    return <div className="flex items-center justify-center h-full text-white/30 text-sm">Owner access required</div>;
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setNotice(null);
    try {
      await createUser({
        username: form.username,
        password: form.password,
        role: form.role,
        permissions: parsePermissions(form.permissionsText),
      });
      setForm({ username: "", password: "", role: "deployer", permissionsText: "" });
      setNotice({ tone: "ok", text: "User created." });
      await load();
    } catch (err) {
      setNotice({ tone: "danger", text: err instanceof Error ? err.message : "Failed to create user" });
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveUser(user: User) {
    const draft = drafts[user.id];
    if (!draft) return;
    setDrafts((current) => ({ ...current, [user.id]: { ...draft, saving: true } }));
    try {
      await updateUser(user.id, {
        role: draft.role,
        permissions: parsePermissions(draft.permissionsText),
      });
      await load();
      setNotice({ tone: "ok", text: `Updated ${user.username}.` });
    } catch (err) {
      setNotice({ tone: "danger", text: err instanceof Error ? err.message : "Failed to update user" });
      setDrafts((current) => ({ ...current, [user.id]: { ...draft, saving: false } }));
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this user?")) return;
    try {
      await deleteUser(id);
      await load();
    } catch (err) {
      setNotice({ tone: "danger", text: err instanceof Error ? err.message : "Failed to delete user" });
    }
  }

  return (
    <div className="space-y-5">
      <div>
        <div className="eyebrow mb-0.5">Team</div>
        <h1 className="text-xl font-semibold text-white">User management</h1>
      </div>

      {notice && (
        <div className={cn("rounded px-3 py-2 text-sm border", notice.tone === "ok" ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : "bg-red-500/10 border-red-500/30 text-red-400")}>
          {notice.text}
        </div>
      )}

      <div className="space-y-3">
        {users.map((user) => {
          const draft = drafts[user.id] ?? { role: user.role, permissionsText: formatPermissions(user.permissions ?? []) };
          return (
            <div key={user.id} className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5 space-y-4">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="text-sm font-medium text-white">{user.username}</div>
                  <div className="text-xs text-white/35">{user.role}</div>
                </div>
                <button
                  type="button"
                  onClick={() => handleDelete(user.id)}
                  disabled={user.username === currentUser?.username}
                  className="text-xs text-white/40 hover:text-red-400 transition-colors px-2 py-1 rounded hover:bg-white/[0.04] disabled:opacity-30 disabled:cursor-not-allowed"
                >
                  Remove
                </button>
              </div>

              <div className="grid grid-cols-1 lg:grid-cols-[180px_1fr] gap-4">
                <Field label="Global role">
                  <select
                    value={draft.role}
                    onChange={(e) => setDrafts((current) => ({ ...current, [user.id]: { ...draft, role: e.target.value } }))}
                    className="text-input"
                  >
                    <option value="owner">owner</option>
                    <option value="admin">admin</option>
                    <option value="deployer">deployer</option>
                    <option value="viewer">viewer</option>
                  </select>
                </Field>
                <Field label="Scoped access">
                  <textarea
                    className="text-input min-h-[110px] resize-y font-mono text-xs"
                    value={draft.permissionsText}
                    onChange={(e) => setDrafts((current) => ({ ...current, [user.id]: { ...draft, permissionsText: e.target.value } }))}
                    placeholder={"my-app staging deployer\nmy-app prod viewer\n* preview viewer"}
                  />
                </Field>
              </div>

              <div className="flex items-center justify-between gap-3">
                <div className="text-xs text-white/35">
                  One rule per line: <code>app env role</code>. Use <code>*</code> for all apps or all envs.
                </div>
                <button
                  type="button"
                  onClick={() => void handleSaveUser(user)}
                  disabled={draft.saving}
                  className="text-sm bg-white text-black font-semibold px-4 py-2 rounded hover:bg-white/90 transition-colors disabled:opacity-40"
                >
                  {draft.saving ? "Saving..." : "Save access"}
                </button>
              </div>
            </div>
          );
        })}
      </div>

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
                <option value="admin">admin</option>
                <option value="deployer">deployer</option>
                <option value="viewer">viewer</option>
              </select>
            </Field>
          </div>

          <Field label="Scoped access">
            <textarea
              className="text-input min-h-[110px] resize-y font-mono text-xs"
              value={form.permissionsText}
              onChange={(e) => setForm((f) => ({ ...f, permissionsText: e.target.value }))}
              placeholder={"my-app staging deployer\nmy-app prod viewer\n* preview viewer"}
            />
          </Field>

          <button type="submit" disabled={busy} className="text-sm bg-white text-black font-semibold px-4 py-2 rounded hover:bg-white/90 transition-colors disabled:opacity-40">
            {busy ? "Creating..." : "Create user"}
          </button>
        </form>
      </div>
    </div>
  );
}

function formatPermissions(permissions: UserPermission[]): string {
  return (permissions ?? [])
    .map((permission) => `${permission.app} ${permission.env} ${permission.role}`)
    .join("\n");
}

function parsePermissions(text: string): UserPermission[] {
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [app = "", env = "", role = "viewer"] = line.split(/\s+/);
      return { app, env, role };
    })
    .filter((permission) => permission.app && permission.env && permission.role);
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="text-xs text-white/40 mb-1.5">{label}</div>
      {children}
    </label>
  );
}
