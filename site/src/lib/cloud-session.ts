import { cookies } from "next/headers";

export type CloudSession = {
  name: string;
  email: string;
  role: string;
};

export const CLOUD_SESSION_COOKIE = "relay_cloud_session";

function isValidSession(value: unknown): value is CloudSession {
  if (!value || typeof value !== "object") {
    return false;
  }

  const record = value as Record<string, unknown>;
  return (
    typeof record.name === "string" &&
    record.name.length > 0 &&
    typeof record.email === "string" &&
    record.email.length > 0 &&
    typeof record.role === "string" &&
    record.role.length > 0
  );
}

export function encodeCloudSession(session: CloudSession) {
  return Buffer.from(JSON.stringify(session), "utf8").toString("base64url");
}

export function decodeCloudSession(value: string | undefined) {
  if (!value) {
    return null;
  }

  try {
    const parsed = JSON.parse(
      Buffer.from(value, "base64url").toString("utf8"),
    ) as unknown;
    return isValidSession(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

export async function getCloudSession() {
  const store = await cookies();
  return decodeCloudSession(store.get(CLOUD_SESSION_COOKIE)?.value);
}
