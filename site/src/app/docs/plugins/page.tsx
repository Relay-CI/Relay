import { CodePanel, DocsPageLayout, DocsHero as PageHero, SectionHeading } from "@/components/site-shell";

export const metadata = { title: "Buildpack plugins", alternates: { canonical: "/docs/plugins" } };

export default function PluginsPage() {
  return (
    <DocsPageLayout
      currentPath="/docs/plugins"
      hero={
        <PageHero
          eyebrow="Buildpack plugins"
          title="Buildpack plugins"
          summary="Add framework detection and build plans with JSON plugins installed on the agent. Learn how plugins match projects, how to install them, and how to write your own."
          aside={
            <CodePanel
              label="Example install"
              code={`relay plugin search astro
relay plugin install ./plugins/astro-static.json \\
  --url http://127.0.0.1:8080 \\
  --token YOUR_TOKEN
relay plugin install-url https://example.com/plugins/astro-static.json \\
  --sha256 <hex>

relay plugin list --url http://127.0.0.1:8080 --token YOUR_TOKEN`}
              footer="Install and remove require RELAY_ENABLE_PLUGIN_MUTATIONS=true on the agent. Remote installs are HTTPS-only. Astro remains the sample plugin in this repo."
            />
          }
        />
      }
    >
      <section id="plugin-model" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Plugin model"
          title="A plugin is a detection rule set plus a Docker plan template."
          body="A plugin tells the agent how to recognize a framework, build its image, and start the app."
        />
        <div className="grid gap-5 lg:grid-cols-4">
          {[
            {
              title: "Detection happens on the agent",
              body: "Relay inspects the uploaded repo snapshot on the server. Plugins are applied there, so every deploy for that agent sees the same framework support.",
            },
            {
              title: "Priority decides matching order",
              body: "Plugins are prepended ahead of built-ins. Higher priority wins, so a plugin can override the default behavior for a specific framework shape.",
            },
            {
              title: "Build plan",
              body: "A plugin does not bypass the rest of Relay. It still resolves to a build image, run image, service port, and rendered Dockerfile.",
            },
            {
              title: "Install trust is explicit",
              body: "Remote plugin installs are owner-only, require HTTPS, and can be pinned with SHA256 so the server verifies the exact JSON before it writes anything under data/plugins/buildpacks.",
            },
          ].map((item) => (
            <article key={item.title} className="site-article-card">
              <h3 className="text-xl font-semibold tracking-[0.01em] text-[color:var(--ink)]">
                {item.title}
              </h3>
              <p className="mt-3 text-sm leading-7 text-[color:var(--muted)]">{item.body}</p>
            </article>
          ))}
        </div>
      </section>

      <section id="installing-plugins" className="scroll-mt-28 grid gap-8 lg:grid-cols-[1fr_24rem]">
        <div className="rounded-[2rem] border border-[color:var(--line)] bg-white/70 p-7">
          <SectionHeading
            label="Installing plugins"
            title="Install and remove plugins"
            body="A plugin install is an authenticated API write to the agent. In production, leave plugin mutation off unless you are actively curating framework support."
          />
          <div className="mt-7 grid gap-4">
            {[
              ["1", "Turn on mutation intentionally", "Set `RELAY_ENABLE_PLUGIN_MUTATIONS=true` only while administering plugins."],
              ["2", "Choose a source", "Install from a JSON file, paste JSON in the owner dashboard, use catalog search, or use `relay plugin install-url <https-url> --sha256 <hex>` for a verified remote source."],
              ["3", "List what is active", "Use `relay plugin list`, the owner dashboard Plugins tab, or `GET /api/plugins/buildpacks` to inspect the live set."],
              ["4", "Turn mutation back off", "Once the plugin is installed, the agent still loads it even after you disable further mutations."],
            ].map(([step, title, body]) => (
              <div
                key={step}
                className="grid gap-3 rounded-[1.4rem] border border-[color:var(--line)] bg-[color:var(--paper)] p-5 sm:grid-cols-[56px_1fr]"
              >
                <div className="text-sm font-semibold tracking-[0.28em] text-[color:var(--accent)]">{step}</div>
                <div>
                  <h3 className="text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">{title}</h3>
                  <p className="mt-1 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <CodePanel
          label="Plugin API"
          code={`GET    /api/plugins/buildpacks
POST   /api/plugins/buildpacks
POST   /api/plugins/buildpacks/install-url
DELETE /api/plugins/buildpacks/:name`}
          footer="Plugin definitions live under data/plugins/buildpacks on the server."
        />
      </section>

      <section id="authoring-plugins" className="scroll-mt-28 grid gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <div className="space-y-5">
          <SectionHeading
            label="Authoring plugins"
            title="Write the plugin around file signals and a concrete build/run plan."
            body="A good plugin is narrow. Match on a specific framework shape, render a Dockerfile that is easy to reason about, and avoid shadowing unrelated repos with broad rules."
          />
          <div className="grid gap-4">
            {[
              ["name", "Unique plugin id used for storage and listing."],
              ["priority", "Higher numbers match before lower ones."],
              ["detect", "Files, directories, extensions, or package.json signals that identify the framework."],
              ["plan", "Kind, images, service port, commands, and the Dockerfile template."],
            ].map(([name, body]) => (
              <div key={name} className="rounded-[1.5rem] border border-[color:var(--line)] bg-white/75 px-5 py-4">
                <div className="font-mono text-sm font-semibold text-[color:var(--ink)]">{name}</div>
                <p className="mt-1 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
              </div>
            ))}
          </div>
        </div>

        <CodePanel
          label="Plugin shape"
          code={"{\n  \"name\": \"astro-static\",\n  \"priority\": 900,\n  \"detect\": {\n    \"files_any\": [\"astro.config.mjs\", \"astro.config.ts\"],\n    \"package_deps_any\": [\"astro\"]\n  },\n  \"plan\": {\n    \"kind\": \"astro-static\",\n    \"service_port\": 80,\n    \"dockerfile_template\": \"FROM node:22 AS builder\\n...\\nFROM nginx:alpine\\n...\"\n  }\n}"}
          footer="Sprint UI is built in now. plugins/astro-static.json remains the sample plugin for new framework support."
        />
      </section>

      <section id="built-in-buildpacks" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="Built-in buildpacks"
          title="Built-ins cover the common stacks before custom plugins fill the gaps."
          body="Built-ins cover the most common stacks. Plugins run first and can override detection for any of them. If detection lands on a built-in, the agent renders its Dockerfile and runs the build with no additional config."
        />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[
            ["node-next", "Next.js apps detect next in package.json and build with next build."],
            ["node-vite", "Vite-based SPAs match vite in package.json and output a static dist/."],
            ["expo-web", "Expo web targets detect expo in package.json and run web builds."],
            ["sprint-ui", "Sprint UI apps detect Sprint UI signals and render the expected web build plan."],
            ["bun", "Bun apps detect bun.lock or Bun package usage and build with Bun-aware commands."],
            ["node-generic", "Any Node.js project with a start script and no framework-specific signal."],
            ["go", "Go modules detect go.mod, cross-compile, and run the resulting binary."],
            ["dotnet", "ASP.NET Core apps detect .csproj and build with dotnet publish."],
            ["python", "Python apps, including Flask and FastAPI, detect pyproject.toml or requirements.txt."],
            ["ruby", "Rails and Rack apps detect Gemfile and run the Ruby build and boot path."],
            ["java", "Maven or Gradle projects detect pom.xml or build.gradle and package a JAR."],
            ["rust", "Rust binaries detect Cargo.toml and build in release mode."],
            ["c-cpp", "C and C++ projects detect a Makefile and compile with make."],
            ["wasm-static", "WebAssembly targets compiled to static files detect wasm-pack or similar."],
            ["static", "Pure static sites use any directory with an index.html and no build step."],
            ["dockerfile", "An explicit repo-relative Dockerfile or an auto-detected nested Dockerfile under the selected app roots overrides all detection."],
          ].map(([name, body]) => (
            <div
              key={name}
              className="rounded-[1.5rem] border border-[color:var(--line)] bg-[color:var(--paper)] px-5 py-4"
            >
              <div className="font-mono text-sm font-semibold text-[color:var(--accent-strong)]">{name}</div>
              <p className="mt-1 text-sm leading-6 text-[color:var(--muted)]">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section id="what-comes-next" className="scroll-mt-28 space-y-8">
        <SectionHeading
          label="What comes next"
          title="Planned plugin features"
          body="These are planned features. They are not part of the current buildpack plugin API."
        />
        <div className="grid gap-5 lg:grid-cols-4">
          {[
            {
              title: "CI step plugins",
              body: "relay ci test and relay ci lint would run typed check steps against the uploaded snapshot before the build starts. Plugins supply the command, image, and pass/fail rule.",
            },
            {
              title: "Versioned packages",
              body: "Plugin search is available. Version tracking, upgrades, and a signed release channel are planned.",
            },
            {
              title: "Signed catalogs and trust policy",
              body: "HTTPS and optional SHA256 cover transport and integrity, but Relay still lacks catalog signatures, allowlists, and richer trust policy for remote plugin sources.",
            },
            {
              title: "Cross-compile artifacts",
              body: "relay build --target linux-amd64 would separate the build artifact from the image, enabling multi-arch deployments and pre-built binaries without re-running the full build for each target.",
            },
          ].map((item) => (
            <article key={item.title} className="site-article-card">
              <h3 className="text-xl font-semibold tracking-[0.01em] text-[color:var(--ink)]">
                {item.title}
              </h3>
              <p className="mt-3 text-sm leading-7 text-[color:var(--muted)]">{item.body}</p>
            </article>
          ))}
        </div>
      </section>
    </DocsPageLayout>
  );
}
