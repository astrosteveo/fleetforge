# FleetForge

[![Powered by Awesome Copilot](https://img.shields.io/badge/Powered_by-Awesome_Copilot-blue?logo=githubcopilot)](https://aka.ms/awesome-github-copilot)

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
Access prompts in GitHub Copilot Chat using the `/` command:
```
/create-implementation-plan
/breakdown-feature-implementation
/sql-optimization
/multi-stage-dockerfile
```

### Using Chat Modes
Activate specialized AI assistants:
```
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

```
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