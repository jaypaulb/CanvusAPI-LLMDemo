# Backend Testing Standard

## 1. Atomic Design Context

Backend testing focuses on database and API organism layers. Tests themselves are NOT production code—they validate organisms, molecules, and atoms.

**Testing Hierarchy:**
- **Atom tests**: Pure validation/formatting functions (NO mocks, NO database, NO external dependencies)
- **Molecule tests**: Composite helpers (minimal mocking for crypto, Date, random values)
- **Organism tests**: Models and API endpoints (integration tests with test database, real atoms/molecules)
- **Test organism agent**: `test-writer-organism` generates integration tests for organisms

**Key Principle**: Test dependencies flow bottom-up. Atom tests are isolated. Molecule tests may mock side effects. Organism tests use real database and compose real atoms/molecules.

## 2. When to Use This Skill

Claude Code should apply backend testing standards when encountering:

- "I need to test the [backend feature]"
- "Write tests for the [model/API endpoint]"
- "Add integration tests for [service]"
- "Test the [database operation]"
- "Create test coverage for [controller]"
- "Verify [validation logic] works correctly"
- "Test database associations for [model]"
- "Add API endpoint tests for [resource]"

## 3. BV/BD Integration Patterns

When working in beads mode, integrate testing into your workflow:

```bash
# Find backend test gaps
bv --recipe actionable --filter "backend" --filter "test-gap"

# Check test dependencies before writing
bd show bd-test-123 --deps  # Should depend on model/API bead

# After writing tests, close test bead
bd close bd-test-123 --comment "Added integration tests at tests/organisms/UserModel.test.js"

# Verify test coverage for high-impact organisms
bv --recipe high-impact --format json | jq '.[] | select(.tags[]? == "needs-tests")'
```

**Integration Checklist:**
- Test beads depend on organism beads (models, APIs), never vice versa
- Organism completion should trigger test bead creation
- Test failures should block bead closure
- Tag test beads: `test-organism`, `backend`, `[model-or-api-name]`

## 4. Dependency Rules

**Atom tests:**
- NO external dependencies (database, API, filesystem)
- NO mocks (pure functions = predictable outputs)
- Test with direct inputs/outputs only

**Molecule tests:**
- Minimal mocks (Date.now, crypto.randomBytes, external APIs)
- Real atoms (call validators, formatters directly)
- NO database or filesystem

**Organism tests:**
- Test database (in-memory or isolated test DB)
- Real atoms and molecules (no mocking internal code)
- Mock external services (Stripe, SendGrid, AWS)
- Integration test approach (full stack from request to database)

## 5. Testing Strategy

**Unit Tests (Model Methods & Validations):**
- Model instance methods return expected values
- Validations fire and reject invalid data
- Default values are set correctly
- Computed properties calculate accurately

**Integration Tests (Database Operations):**
- Models save to and read from test database
- Associations load correctly (belongsTo, hasMany)
- Cascade behaviors work (delete, nullify)
- Constraints are enforced (unique, foreign keys)

**Integration Tests (API Requests/Responses):**
- Full request/response cycle (HTTP method → controller → model → response)
- Authentication middleware blocks unauthorized requests
- Response formats match schema (status codes, JSON structure)
- Error handling returns appropriate codes and messages

**Mock External APIs:**
- Stripe payment processing
- SendGrid email delivery
- AWS S3 file uploads
- Third-party API integrations

**Use Test Database:**
- Separate test DB or in-memory database
- Reset state between tests (transactions or truncate)
- Seed minimal fixtures per test
- Avoid shared state across tests

## 6. Code Examples

### Example 1: Model Integration Test (User.create with Test DB)

```javascript
// tests/organisms/UserModel.test.js
import { User } from '../../models/User.js';
import { setupTestDB, teardownTestDB } from '../helpers/testDatabase.js';

describe('User Model', () => {
  beforeAll(async () => await setupTestDB());
  afterAll(async () => await teardownTestDB());
  afterEach(async () => await User.destroy({ where: {}, truncate: true }));

  it('creates user with valid email and hashed password', async () => {
    const user = await User.create({
      email: 'test@example.com',
      password: 'SecurePass123'
    });

    expect(user.id).toBeDefined();
    expect(user.email).toBe('test@example.com');
    expect(user.password_hash).toBeDefined();
    expect(user.password_hash).not.toBe('SecurePass123'); // Password is hashed
  });

  it('rejects duplicate email addresses', async () => {
    await User.create({ email: 'duplicate@example.com', password: 'Pass123' });

    await expect(
      User.create({ email: 'duplicate@example.com', password: 'Pass456' })
    ).rejects.toThrow(/unique constraint/i);
  });

  it('loads posts association correctly', async () => {
    const user = await User.create({ email: 'author@example.com', password: 'Pass123' });
    await user.createPost({ title: 'First Post', content: 'Hello world' });

    const userWithPosts = await User.findByPk(user.id, { include: 'posts' });
    expect(userWithPosts.posts).toHaveLength(1);
    expect(userWithPosts.posts[0].title).toBe('First Post');
  });
});
```

### Example 2: API Endpoint Test (POST /users with Supertest)

```javascript
// tests/organisms/UserAPI.test.js
import request from 'supertest';
import { app } from '../../app.js';
import { User } from '../../models/User.js';
import { setupTestDB, teardownTestDB } from '../helpers/testDatabase.js';

describe('POST /api/users', () => {
  beforeAll(async () => await setupTestDB());
  afterAll(async () => await teardownTestDB());
  afterEach(async () => await User.destroy({ where: {}, truncate: true }));

  it('creates user with valid data and returns 201', async () => {
    const response = await request(app)
      .post('/api/users')
      .send({ email: 'newuser@example.com', password: 'SecurePass123' });

    expect(response.status).toBe(201);
    expect(response.body.data.email).toBe('newuser@example.com');
    expect(response.body.data.password_hash).toBeUndefined(); // Don't expose hash

    const user = await User.findOne({ where: { email: 'newuser@example.com' } });
    expect(user).toBeDefined();
  });

  it('rejects invalid email with 400', async () => {
    const response = await request(app)
      .post('/api/users')
      .send({ email: 'invalid-email', password: 'SecurePass123' });

    expect(response.status).toBe(400);
    expect(response.body.errors.email).toBeDefined();
  });

  it('requires authentication token for protected routes', async () => {
    const response = await request(app)
      .get('/api/users/me')
      .set('Authorization', ''); // No token

    expect(response.status).toBe(401);
    expect(response.body.error).toMatch(/authentication required/i);
  });
});
```

### Example 3: Molecule Test (Validator with Minimal Mocking)

```javascript
// tests/molecules/passwordHandler.test.js
import { PasswordHandler } from '../../molecules/PasswordHandler.js';

describe('PasswordHandler', () => {
  it('hashes password and verifies correctly', () => {
    const plaintext = 'MySecurePassword123';
    const hashed = PasswordHandler.prepare(plaintext);

    expect(hashed).not.toBe(plaintext);
    expect(PasswordHandler.check(plaintext, hashed)).toBe(true);
    expect(PasswordHandler.check('WrongPassword', hashed)).toBe(false);
  });

  it('rejects weak passwords', () => {
    expect(() => PasswordHandler.prepare('weak')).toThrow(/too weak/i);
  });
});
```

## Verification Checklist

After writing backend tests:

- [ ] Atom tests have NO mocks or external dependencies
- [ ] Molecule tests mock only external side effects (Date, crypto, APIs)
- [ ] Organism tests use test database and real atoms/molecules
- [ ] API tests cover full request/response cycle with supertest
- [ ] Model tests verify database operations and associations
- [ ] External services (Stripe, SendGrid) are mocked
- [ ] Test database is reset between tests (no shared state)
- [ ] All tests pass before marking bead complete
- [ ] BV/BD test bead closed with comment pointing to test files
- [ ] No production code depends on test organisms
