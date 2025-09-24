# FleetForge - Awesome Copilot Demo

This file demonstrates how to use the awesome-copilot features installed in FleetForge.

## 🚀 Quick Start Examples

### Project Management & Planning

1. **Create a new implementation plan**:
   ```
   /create-implementation-plan
   ```

2. **Break down a feature into tasks**:
   ```
   /breakdown-feature-implementation
   ```

3. **Plan technical spikes**:
   ```
   /create-technical-spike
   ```

### Database Operations

1. **Get PostgreSQL optimization advice**:
   ```
   /postgresql-optimization
   ```

2. **Review SQL code**:
   ```
   /sql-code-review
   ```

3. **Get expert DBA help**:
   ```
   @postgresql-dba How do I optimize this query for performance?
   @ms-sql-dba What's the best way to handle this stored procedure?
   ```

### Containerization & DevOps

1. **Create optimized Dockerfiles**:
   ```
   /multi-stage-dockerfile
   ```

2. **Get Azure architecture guidance**:
   ```
   @azure-principal-architect Design a scalable architecture for this application
   ```

## 🔧 Automatic Features

The following happen automatically when you work with relevant files:

- **Dockerfile**: Docker best practices are automatically applied
- **SQL files**: SQL generation standards and optimization guidelines
- **Any development task**: Task implementation workflow is followed
- **All files**: Specification-driven development workflow

## 📁 File Structure

```
.github/
├── prompts/14          # Use with /prompt-name
├── instructions/6      # Auto-applied to relevant files  
├── chatmodes/10        # Use with @mode-name
└── copilot-instructions.md  # Repository-level instructions
```

## 💡 Pro Tips

1. **Use tab completion** - Type `/` or `@` to see available options
2. **Combine modes** - Use chat modes for context, then prompts for specific tasks
3. **Check instructions** - Instructions automatically improve code quality
4. **Repository instructions** - GitHub Copilot uses `.github/copilot-instructions.md` automatically

## 🆘 Getting Help

- List all available prompts: Check `.github/prompts/` directory
- List all chat modes: Check `.github/chatmodes/` directory  
- View configuration: `awesome-copilot.config.yml`
- Modify setup: Edit config and re-run installation steps in README