# Test Writing Standard

## 1. Atomic Design Context

Testing strategy aligns with atomic design principles to create a robust test pyramid:

**Test Pyramid by Atomic Level:**
- **Atoms** → Unit tests (pure function testing, no mocks)
  - Fast, isolated, deterministic
  - 100% coverage target
  - Test pure logic, edge cases, error handling
  - Examples: validators, formatters, calculators

- **Molecules** → Unit tests with minimal mocking (2-3 atom interactions)
  - Test atom composition and interaction logic
  - Mock only when atoms have unavoidable side effects
  - Examples: password hasher, email sender, data transformer

- **Organisms** → Integration tests (database, API, state management)
  - Mock external dependencies (third-party APIs, file systems)
  - Use real atoms and molecules
  - Test database interactions with test DB
  - Examples: User model, AuthService, OrderProcessor

- **Pages/Integration** → E2E tests (full user flows)
  - Real everything in staging environment
  - Test cross-organism interactions
  - Minimal tests (most expensive)
  - Examples: registration flow, checkout process

**Test Pyramid Distribution:**
- Many atom tests (fast, cheap, comprehensive)
- Fewer organism tests (slower, integration-focused)
- Minimal E2E tests (slowest, most expensive, critical paths only)

This ensures fast feedback loops while maintaining confidence in system behavior.

## 2. When to Use This Skill

Claude Code should recognize test-writing context from these patterns:

**User Intent Signals:**
- "I need to write tests for [component/function/module]"
- "I'm adding test coverage for [feature]"
- "I need to test [interaction/behavior]"
- "I'm writing integration tests for [organism]"
- "Write E2E tests for [user flow]"
- "Add unit tests for [atom/molecule]"

**Agent Selection:**
- Use `test-writer-molecule` agent for molecule-level tests (2-3 atom compositions)
- Use `test-writer-organism` agent for organism-level tests (integration tests)
- Use general test-writing skill for atom tests (pure functions)
- Use E2E testing tools (Playwright/Cypress) for page/integration tests

**When to Write Tests:**
- After completing feature implementation (not during intermediate steps)
- At logical completion points (feature done, ready to verify)
- When adding critical user flows (registration, checkout, etc.)
- When test-gap-analyzer identifies missing coverage

## 3. BV/BD Integration Patterns

When working in beads mode, integrate testing with issue tracking:

```bash
# Find untested issues (gap analysis)
bv --recipe actionable --format json | jq '.[] | select(.tags[]? == "test-gap")'

# Check test coverage gaps
bd list --tag untested --format json

# After writing tests, verify test issues closed
bd show bd-601 --status  # Verify test issue closed

# Use test-gap-analyzer agent to identify missing coverage
# Run after organisms implemented to find integration test gaps
```

**Workflow Integration:**
1. Implement feature/organism
2. Run `test-gap-analyzer` to identify missing tests
3. Check for existing test-gap beads: `bd list --tag test-gap`
4. Write tests for identified gaps
5. Close test-gap beads after verification
6. Update bead with test file locations

**Tagging Convention:**
- `test-gap`: Missing test coverage identified
- `untested`: Code without any tests
- `atom-test`: Unit tests for atoms
- `organism-test`: Integration tests for organisms
- `e2e-test`: End-to-end test coverage

## 4. Dependency Rules

Test layer rules enforce proper isolation and realistic integration:

**Atom Tests:**
- NO external dependencies (pure function testing)
- NO mocks (defeats purpose of atom purity)
- NO database, file system, API calls
- Example: `expect(validateEmail('test@example.com')).toBe(true)`

**Molecule Tests:**
- Can mock atoms if needed, but prefer real atoms
- Mock only unavoidable side effects (crypto.randomBytes, Date.now)
- Example: Mock random salt generation, use real hash function

**Organism Tests:**
- Mock external dependencies (third-party APIs, external services)
- Use real atoms and molecules
- Use test database (in-memory or isolated test DB)
- Example: Mock Stripe API, use real User model with test DB

**Integration Tests:**
- Use test database (isolated from production)
- Real API (localhost/staging)
- Real atoms, molecules, organisms
- Example: POST /users with real database and validators

**E2E Tests:**
- Real everything (staging environment)
- No mocks
- Full system integration
- Example: Browser automation through complete user registration flow

## 5. Testing Strategy

### Atom Testing Strategy
- **Goal:** 100% coverage, fast execution (milliseconds)
- **Tools:** Jest, Vitest, pytest (no external dependencies)
- **Focus:** Pure logic, edge cases, error handling
- **No mocks:** Atoms should be pure functions
- **Examples:** Email validator, price calculator, string formatter

### Molecule Testing Strategy
- **Goal:** Test atom composition and interaction
- **Tools:** Jest + minimal mocks (crypto, Date)
- **Focus:** How atoms work together
- **Selective mocking:** Only unavoidable side effects
- **Examples:** Password hasher (hash + salt), email sender (validator + SMTP)

### Organism Testing Strategy
- **Goal:** Integration testing with real atoms/molecules
- **Tools:** Jest + test DB + API mocks
- **Focus:** Database interactions, business logic, state management
- **Mock externals:** Third-party APIs, file systems, payment gateways
- **Examples:** User model (CRUD operations), AuthService (login/logout)

### Page/Integration Testing Strategy
- **Goal:** Critical user flows only (expensive tests)
- **Tools:** Playwright, Cypress, Selenium
- **Focus:** Full-stack user workflows
- **Real everything:** Staging database, real APIs, real services
- **Examples:** User registration, checkout flow, admin dashboard

## 6. Code Examples

### Atom Test: Email Validator (Pure Function)

```javascript
// src/atoms/validateEmail.test.js
const { validateEmail } = require('./validateEmail');

describe('validateEmail (atom)', () => {
  it('returns true for valid email', () => {
    expect(validateEmail('user@example.com')).toBe(true);
  });

  it('returns false for missing @', () => {
    expect(validateEmail('userexample.com')).toBe(false);
  });

  it('returns false for missing domain', () => {
    expect(validateEmail('user@')).toBe(false);
  });

  it('returns false for empty string', () => {
    expect(validateEmail('')).toBe(false);
  });

  it('returns false for null', () => {
    expect(validateEmail(null)).toBe(false);
  });
});
// NO MOCKS - pure function testing
```

### Molecule Test: Password Hasher (Atom Composition)

```javascript
// src/molecules/hashPassword.test.js
const { hashPassword, verifyPassword } = require('./hashPassword');
const crypto = require('crypto');

// Mock only unavoidable side effects
jest.mock('crypto', () => ({
  randomBytes: jest.fn(() => Buffer.from('fixed-salt')),
  pbkdf2Sync: jest.requireActual('crypto').pbkdf2Sync
}));

describe('hashPassword (molecule)', () => {
  it('hashes password with salt', () => {
    const hashed = hashPassword('mypassword');
    expect(hashed).toMatch(/^fixed-salt:/);
  });

  it('verifyPassword returns true for correct password', () => {
    const hashed = hashPassword('mypassword');
    expect(verifyPassword('mypassword', hashed)).toBe(true);
  });

  it('verifyPassword returns false for incorrect password', () => {
    const hashed = hashPassword('mypassword');
    expect(verifyPassword('wrongpassword', hashed)).toBe(false);
  });
});
// Minimal mocking - only crypto.randomBytes for deterministic tests
```

### Organism Test: User Model (Database Integration)

```javascript
// src/organisms/User.test.js
const { User } = require('./User');
const { setupTestDB, teardownTestDB } = require('../../test/helpers/db');

describe('User model (organism)', () => {
  beforeAll(async () => {
    await setupTestDB(); // In-memory SQLite or isolated test DB
  });

  afterAll(async () => {
    await teardownTestDB();
  });

  it('creates user with valid email', async () => {
    const user = await User.create({
      email: 'test@example.com',
      password: 'securepass123'
    });
    expect(user.id).toBeDefined();
    expect(user.email).toBe('test@example.com');
  });

  it('throws error for duplicate email', async () => {
    await User.create({ email: 'dup@example.com', password: 'pass' });
    await expect(
      User.create({ email: 'dup@example.com', password: 'pass2' })
    ).rejects.toThrow('Email already exists');
  });
});
// Real database (test DB), real validators, no external API mocks
```

### API Organism Test: POST /users (Request/Response)

```javascript
// src/api/users.test.js
const request = require('supertest');
const app = require('../app');
const { setupTestDB, teardownTestDB } = require('../../test/helpers/db');

describe('POST /users (organism)', () => {
  beforeAll(async () => {
    await setupTestDB();
  });

  afterAll(async () => {
    await teardownTestDB();
  });

  it('creates user and returns 201', async () => {
    const response = await request(app)
      .post('/users')
      .send({ email: 'new@example.com', password: 'securepass' });

    expect(response.status).toBe(201);
    expect(response.body.email).toBe('new@example.com');
    expect(response.body.password).toBeUndefined(); // Not exposed
  });

  it('returns 400 for invalid email', async () => {
    const response = await request(app)
      .post('/users')
      .send({ email: 'invalid', password: 'securepass' });

    expect(response.status).toBe(400);
    expect(response.body.error).toMatch(/email/i);
  });
});
// Real API, real database, real validators
```

### E2E Test: User Registration Flow (Full Stack)

```javascript
// tests/e2e/registration.spec.js
const { test, expect } = require('@playwright/test');

test.describe('User Registration (E2E)', () => {
  test('completes full registration flow', async ({ page }) => {
    // Navigate to registration page
    await page.goto('http://localhost:3000/register');

    // Fill form
    await page.fill('[name="email"]', 'e2e@example.com');
    await page.fill('[name="password"]', 'SecurePass123');
    await page.fill('[name="confirmPassword"]', 'SecurePass123');

    // Submit
    await page.click('button[type="submit"]');

    // Verify success redirect
    await expect(page).toHaveURL('http://localhost:3000/dashboard');
    await expect(page.locator('.welcome-message')).toContainText('Welcome');
  });

  test('shows error for existing email', async ({ page }) => {
    await page.goto('http://localhost:3000/register');
    await page.fill('[name="email"]', 'existing@example.com');
    await page.fill('[name="password"]', 'SecurePass123');
    await page.fill('[name="confirmPassword"]', 'SecurePass123');
    await page.click('button[type="submit"]');

    await expect(page.locator('.error-message')).toContainText('Email already exists');
  });
});
// Real browser, real database (staging), real API, full user flow
```

### Test Gap Analysis Example

```bash
# Use test-gap-analyzer to find missing coverage
# After implementing organisms, run:

$ test-gap-analyzer --path src/organisms --format json

# Output identifies untested organisms:
{
  "gaps": [
    {
      "file": "src/organisms/OrderProcessor.js",
      "type": "organism",
      "missing_tests": ["integration", "error_handling"],
      "coverage": "0%"
    }
  ]
}

# Create bead for test gap:
$ bd create --title "Add integration tests for OrderProcessor" --tag test-gap --tag organism-test

# Write tests, then close bead:
$ bd close bd-602 --comment "Added integration tests at tests/organisms/OrderProcessor.test.js"
```

## Test Coverage Best Practices

- **Write Minimal Tests During Development**: Do NOT write tests for every change or intermediate step. Focus on completing the feature implementation first, then add strategic tests only at logical completion points
- **Test Only Core User Flows**: Write tests exclusively for critical paths and primary user workflows. Skip writing tests for non-critical utilities and secondary workflows until if/when you're instructed to do so.
- **Defer Edge Case Testing**: Do NOT test edge cases, error states, or validation logic unless they are business-critical. These can be addressed in dedicated testing phases, not during feature development.
- **Test Behavior, Not Implementation**: Focus tests on what the code does, not how it does it, to reduce brittleness
- **Clear Test Names**: Use descriptive names that explain what's being tested and the expected outcome
- **Mock External Dependencies**: Isolate units by mocking databases, APIs, file systems, and other external services (at organism level and above)
- **Fast Execution**: Keep unit tests fast (milliseconds) so developers run them frequently during development
