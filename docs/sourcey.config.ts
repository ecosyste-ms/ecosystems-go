import { defineConfig } from "sourcey";

// API reference site for the ecosyste.ms Go client library.
//
// The godoc snapshot is a generated artifact and is NOT committed — the Docs
// workflow regenerates it from the checked-out source on every run, so the
// published reference can never drift from `main`:
//
//   npx sourcey godoc -m . -o docs/godoc.json   # from the repo root (needs Go)
//   cd docs && npx sourcey build                # renders docs/dist (no Go needed)
//
// Or, locally: `make docs`.
export default defineConfig({
  prettyUrls: "strip",
  baseUrl: "/ecosystems-go",
  name: "ecosystems-go",
  // Source links from every generated symbol back to the Go source on main.
  repo: "https://github.com/ecosyste-ms/ecosystems-go",
  editBranch: "main",
  editBasePath: "",
  theme: {
    preset: "default",
    colors: { primary: "#1f6feb" },
  },
  navigation: {
    tabs: [
      {
        tab: "Guide",
        slug: "",
        groups: [{ group: "Start", pages: ["introduction"] }],
      },
      {
        tab: "API Reference",
        slug: "api",
        godoc: {
          // Config lives in docs/, the Go module is at the repo root.
          module: "..",
          packages: ["./..."],
          snapshot: "godoc.json",
          mode: "snapshot",
          includeTests: true,
          sourceBasePath: "",
        },
      },
    ],
  },
});
