# Docs Site Playbook

This playbook outlines how to build, preview, and customize the FleetForge documentation site powered by MkDocs Material.

## Local Setup

### Prerequisites

```bash
python3 -m venv .venv-docs
source .venv-docs/bin/activate
pip install --upgrade pip
pip install -r requirements.txt
```

> Always work inside `.venv-docs` to mirror the exact toolchain used in CI. Recreate the environment after dependency upgrades.

### Live Preview

```bash
mkdocs serve --watch docs
```

The site is available at <http://localhost:8000>. The `--watch docs` flag ensures edits to Markdown or configuration files trigger an automatic reload.

### Static Build

```bash
mkdocs build --clean --strict
```

The static site is emitted to `site/`. The `--strict` flag matches the CI pipeline: warnings (broken links, malformed refs, accessibility notices) fail the build.

## Deployment

GitHub Actions publishes the site to GitHub Pages when changes land on `main`. The hardened workflow performs:

1. Dependency installation from the pinned `requirements.txt` with pip caching.
2. `mkdocs build --clean --strict` for validation.
3. Upload of the rendered site artifact.
4. Deployment to GitHub Pages via the official `actions/deploy-pages` action.

During pull requests the workflow uploads a downloadable preview artifact (`docs-site-pr`). Reviewers can extract it locally and open `index.html` to validate rendering without a full local build.

## Configuration Checklist

- Update branding, color palette, and features in `mkdocs.yml`.
- Maintain the navigation tree so every Markdown file appears exactly once.
- Register additional Markdown extensions or plugins when new content patterns require them, and pin versions in `requirements.txt`.
- Add custom styling in `docs/stylesheets/extra.css` and reference it through `extra_css`.

## Content Patterns

- Use relative links for cross-references, for example `[Architecture](../architecture/design.md)`.
- Prefer role-based call-to-action cards for landing pages. See the homepage for examples.
- Embed diagrams with Mermaid fences and provide descriptive text nearby for accessibility.
- Highlight critical workflows using Material admonitions (`!!! note`, `!!! warning`).

## Quality Gates

- Run `mkdocs build --clean --strict` locally before opening a pull request.
- Validate external links with `mkdocs build --strict --config-file mkdocs.yml --site-dir site` to ensure caches are rebuilt.
- Lint Markdown with `markdownlint` or the repository's configured tooling.
- Confirm the GitHub Actions status checks (`docs-build` and `pages-build-deployment`) succeed before merging.

## Accessibility Guidelines

- Maintain heading levels without skipping (H2 follows H1, etc.).
- Provide text alternatives for diagrams and images.
- Ensure call-to-action buttons use descriptive link labels.
- Keep color contrast above WCAG 2.2 AA thresholds; use CSS variables supplied by MkDocs Material when possible.
- Verify focus outlines remain visible after any custom CSS changes.

## Extending the Site

- Add new sections by creating Markdown files within the appropriate directory and registering them in `nav`.
- For experimental layouts, create an isolated page under `contributing/` before promoting it to the main navigation.
- Update `docs/stylesheets/extra.css` for global style changes rather than inline HTML.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| Git plugins warn about missing history | Commit new documentation files or ensure the workflow fetch-depth is zero (already handled in CI). |
| Mermaid diagram does not render | Confirm the Mermaid code block is fenced with ` ```mermaid ` and check for syntax errors. |
| Navigation shows duplicate pages | Ensure each page appears once in the `nav` tree and remove conflicting filenames such as `README.md`. |
| CI build reports an accessibility failure | Review the job logs, address the flagged issue, and rerun `mkdocs build --clean --strict` locally. |

For additional guidance, consult the [Material for MkDocs documentation](https://squidfunk.github.io/mkdocs-material/).
