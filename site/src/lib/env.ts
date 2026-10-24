function requireEnv(name: string) {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }

  return value;
}

function requireUrlEnv(name: string) {
  const value = requireEnv(name);

  try {
    return new URL(value).toString().replace(/\/$/, "");
  } catch {
    throw new Error(`Environment variable ${name} must be a valid absolute URL`);
  }
}

export const siteEnv = {
  get siteUrl() { return requireUrlEnv("NEXT_PUBLIC_SITE_URL"); },
  get cloudSiteUrl() { return requireUrlEnv("NEXT_PUBLIC_CLOUD_SITE_URL"); },
  get databaseUrl() { return requireEnv("DATABASE_URL"); },
  get contactEmail() { return process.env.NEXT_PUBLIC_CONTACT_EMAIL?.trim() || null; },
};
