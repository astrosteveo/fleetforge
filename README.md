# FleetForge

[![Powered by Awesome Copilot](https://img.shields.io/badge/Powered_by-Awesome_Copilot-blue?logo=githubcopilot)](https://aka.ms/awesome-copilot)

FleetForge project enhanced with GitHub Copilot customizations for improved development experience.

## 🤖 Awesome Copilot Integration

This project uses [awesome-copilot](https://github.com/astrosteveo/awesome-copilot) to enhance GitHub Copilot with specialized prompts, instructions, and chat modes for:

### 📋 Project Management & Planning

- **Task Planning**: Use chat modes like `/task-planner` and `/planner` for structured project planning
- **Implementation Planning**: Create detailed implementation plans with `/create-implementation-plan`
- **Epic & Feature Breakdown**: Break down large features using prompts like `/breakdown-epic-arch` and `/breakdown-feature-implementation`
- **Technical Spikes**: Research and document technical decisions with `/create-technical-spike`

### 🗄️ Database & Data Management

- **Database Administration**: Expert PostgreSQL and SQL Server assistance with `/postgresql-dba` and `/ms-sql-dba` chat modes
- **SQL Optimization**: Optimize queries and review SQL code with `/sql-optimization` and `/sql-code-review`
- **PostgreSQL Specific**: PostgreSQL optimization and code review with specialized prompts

### 🐳 Containerization & DevOps

- **Docker Best Practices**: Automated Docker optimization following containerization best practices
- **Multi-stage Dockerfiles**: Create efficient Docker builds with `/multi-stage-dockerfile`
- **DevOps Principles**: Core DevOps practices integrated into development workflow
- **Azure DevOps**: Azure-specific guidance with `/azure-principal-architect`

## 🚀 Getting Started

### Using Prompts

```text
/create-implementation-plan
/breakdown-feature-implementation
/sql-optimization
/multi-stage-dockerfile
```

### Using Chat Modes

```text
@task-planner - For project planning and task breakdown
@postgresql-dba - For PostgreSQL database assistance  
@azure-principal-architect - For Azure cloud architecture guidance
```

### Automatic Instructions

The following instructions are automatically applied to relevant file types:

- **Containerization best practices** for Dockerfile and docker-compose files
- **SQL generation standards** for SQL files and stored procedures
- **Task implementation workflow** for development tasks
- **Specification-driven development** workflow

## 📁 Project Structure

```text
.github/
├── prompts/           # Task-specific prompts (14 enabled)
├── instructions/      # Coding standards & best practices (6 enabled)  
├── chatmodes/         # AI personas & specialized modes (10 enabled)
└── copilot-instructions.md  # Repository-level GitHub Copilot instructions
```

## 🔧 Configuration

This project is configured with the following awesome-copilot collections:

- ✅ **project-planning** - Project management and task planning tools
- ✅ **database-data-management** - Database administration and optimization
- ✅ **devops-oncall** - DevOps practices and containerization

To modify the configuration, edit `awesome-copilot.config.yml` and run:

```bash
node awesome-copilot/awesome-copilot.js apply
```

## 📝 Documentation

Core documentation has been reorganized for clarity:

```text
docs/
  product/        # prd.md, requirements.md, tasks.md
  architecture/   # design.md, future diagrams/
  research/       # academic/reference papers
  adr/            # architecture decision records
  ops/            # runbooks (planned)
```

Generate PDF versions of Markdown files (outputs alongside sources):

### Local Generation

Prerequisites: `pandoc` and a LaTeX engine (`tectonic` recommended or `xelatex`). On macOS you can `brew install pandoc tectonic`; on Linux install via your package manager.

Generate all PDFs:

```bash
make pdfs
```

Force a clean rebuild:

```bash
make clean && make pdfs
```

Outputs are created alongside their source (e.g. `docs/architecture/design.md` -> `docs/architecture/design.pdf`).

### GitHub Action

The workflow `/.github/workflows/docs-pdf.yml` automatically:

- Runs on pushes that modify `docs/*.md`, the workflow file, or the `Makefile`.
- Installs pandoc + tectonic.
- Builds PDFs via `make pdfs`.
- Uploads an artifact named `docs-pdfs` containing all generated PDFs.
- Commits updated PDFs back to `main` when changes are detected.

To manually trigger, use the "Run workflow" button (workflow_dispatch).

### Customization

Adjust Pandoc flags in the `Makefile` (geometry, TOC depth, link colors, etc.). Set a different engine:

```bash
make pdfs PANDOC_ENGINE=xelatex
```

If you prefer a Node-based pipeline with headless Chrome rendering, you can reintroduce the earlier `md-to-pdf` script; current setup intentionally avoids that dependency for simplicity