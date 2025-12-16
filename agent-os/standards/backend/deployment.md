## Deployment standards and conventions

## 1. Atomic Design Context

Deployment orchestrates ALL levels of the atomic design hierarchy. Each level produces different build artifacts:

**Deployment Layer Position:**
- **Level**: Cross-cutting concern (affects atoms through pages)
- **Build outputs by level**:
  - Atoms → Compiled libraries, shared utilities
  - Molecules → Reusable modules, npm packages
  - Organisms → Microservices, API services, background workers
  - Pages → Full applications, entry points
- **Configuration strategy**:
  - Atoms use constants (e.g., `MAX_LENGTH = 100`)
  - Molecules use injected config (e.g., function parameters)
  - Organisms use environment variables (e.g., `process.env.DATABASE_URL`)
  - Pages aggregate all configuration (e.g., load env vars, validate, distribute)

**Key Principle**: Deployment scripts are infrastructure code. They orchestrate but contain NO application logic.

## 2. When to Use This Skill

Claude Code should apply deployment standards when encountering these phrases:

- "Deploy [application/service]"
- "Set up deployment for [feature]"
- "Configure [environment] environment"
- "Add CI/CD pipeline"
- "Create Docker configuration"
- "Set up [production/staging/development] environment"
- "Implement deployment automation"
- "Configure infrastructure for [service]"

## 3. BV/BD Integration Patterns

When working in beads mode, use these patterns to discover and track deployment work:

```bash
# Find deployment-related beads
bd list --tag deployment

# Check if deployment work is actionable (infra code ready)
bv --recipe actionable --filter "infra"

# Find high-impact infrastructure changes
bv --recipe high-impact --format json | jq '.[] | select(.tags[]? == "deployment")'

# After deployment, verify dependencies are satisfied
bd show bd-501 --deps  # Show what this deployment depends on
```

**Tagging Strategy:**
- Tag deployment beads with: `deployment`, `infrastructure`, `[environment-name]`
- Tag CI/CD beads with: `ci-cd`, `automation`, `pipeline`
- Tag configuration beads with: `config`, `env-vars`, `[service-name]`

## 4. Dependency Rules

Deployment infrastructure must follow strict isolation rules:

**Deployment scripts (in /deploy or /scripts):**
- NO dependencies on application code
- CAN depend on: build tools, package managers, deployment platforms
- Live in separate directory (e.g., `/deploy`, `/infrastructure`)

**Build configuration (webpack, vite, tsconfig):**
- CAN read from any application level
- Orchestrates compilation from atoms → organisms → pages
- Declares what to build, not how application works

**CI/CD pipelines:**
- Orchestrate: build, test, deploy
- NO application logic (no business rules, data transformations)
- CAN depend on: test runners, build tools, deployment CLIs

## 5. Testing Strategy

Deployment testing follows a different pyramid than application testing:

**Integration tests in CI** (40% of deployment testing):
- Run full test suite against build artifacts
- Verify service-to-service communication
- Database migrations in isolated environment

**Smoke tests after deployment** (30%):
- Health check endpoints respond
- Critical paths work (e.g., user login, data retrieval)
- External dependencies reachable

**Infrastructure validation** (30%):
- Configuration loads correctly
- Environment variables present and valid
- Secrets accessible, permissions correct

**Rollback procedures tested regularly**:
- Practice rollback in staging monthly
- Document rollback steps
- Verify data integrity after rollback

## 6. Code Examples

### Example 1: Build Script Separating Atom Libraries from Organism Services

```javascript
// /deploy/build.js
// NO application imports - pure orchestration

const { execSync } = require('child_process');
const path = require('path');

const BUILD_ORDER = [
  { level: 'atoms', path: 'src/atoms', output: 'dist/lib' },
  { level: 'molecules', path: 'src/molecules', output: 'dist/modules' },
  { level: 'organisms', path: 'src/organisms', output: 'dist/services' },
  { level: 'pages', path: 'src/pages', output: 'dist/apps' }
];

BUILD_ORDER.forEach(({ level, path, output }) => {
  console.log(`Building ${level}...`);
  execSync(`tsc --project ${path}/tsconfig.json --outDir ${output}`, { stdio: 'inherit' });
});
```

### Example 2: Environment Variable Loading (Atoms vs Organisms)

```javascript
// ATOMS: Use constants (compile-time)
// src/atoms/constants/validation.js
export const MAX_EMAIL_LENGTH = 255;
export const MIN_PASSWORD_LENGTH = 8;

// ORGANISMS: Use process.env (runtime)
// src/organisms/services/database.js
export const connectDatabase = () => {
  const config = {
    host: process.env.DB_HOST,
    port: parseInt(process.env.DB_PORT, 10),
    user: process.env.DB_USER,
    password: process.env.DB_PASSWORD
  };
  return createConnection(config);
};

// PAGES: Aggregate and validate all config
// src/pages/app.js
import { validateEnvVars } from '../molecules/validators/env.js';

const requiredVars = ['DB_HOST', 'DB_PORT', 'JWT_SECRET', 'API_KEY'];
validateEnvVars(requiredVars); // Fail fast if missing

const app = createApp();
app.listen(process.env.PORT || 3000);
```

### Example 3: CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/deploy.yml
# Orchestrates build, test, deploy - NO application logic

name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build artifacts (atoms → pages)
        run: npm run build

      - name: Run integration tests
        run: npm run test:integration
        env:
          DATABASE_URL: ${{ secrets.TEST_DATABASE_URL }}

      - name: Deploy to production
        run: npm run deploy:prod
        env:
          DEPLOY_KEY: ${{ secrets.DEPLOY_KEY }}

      - name: Smoke tests
        run: npm run test:smoke
        env:
          API_URL: https://api.production.com
```
