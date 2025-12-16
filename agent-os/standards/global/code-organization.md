## Code Organization

File organization mirrors atomic design hierarchy to enforce dependency rules and enable scalable architecture.

### Atomic Design Context

File organization should reflect atomic design principles. See `atomic-design.md` for complete hierarchy details.

```
src/
  atoms/
    validators/        # Pure validation functions (email, phone, etc.)
    formatters/        # Pure formatting functions (dates, currency, etc.)
    constants/         # Immutable values, config constants
    utils/            # Single-purpose utilities with no peer dependencies
  molecules/
    validators/        # Compose atom validators (createUserValidator)
    helpers/          # Compose atoms into small services
    parsers/          # Compose atoms for data transformation
  organisms/
    models/           # Database models (compose molecules + atoms)
    controllers/      # API controllers (compose molecules + atoms)
    components/       # UI components with state (compose molecules + atoms)
    services/         # Complex business logic services
  templates/
    interfaces/       # TypeScript interfaces, type definitions
    schemas/          # Data schemas, validation contracts
  pages/
    api/             # API route handlers (composition roots)
    views/           # Full page components (composition roots)
    main.js          # Application entry point
```

### When to Use This Skill

Apply this standard when:
- "Where should I put this [file/function]?"
- "How should I organize [feature]?"
- "Create file structure for [module]"
- Refactoring code to improve maintainability
- Starting a new feature or service
- Resolving circular dependency issues

### BV/BD Integration Patterns

File organization should align with beads issue hierarchy:

```bash
# Query beads to understand where code should live
bd show bd-101 --tag atom        # Atoms go in src/atoms/
bd show bd-201 --tag molecule    # Molecules go in src/molecules/
bd show bd-301 --tag organism    # Organisms go in src/organisms/

# Find work at your atomic level
bd ready                         # Shows unblocked issues
bd list --status in_progress     # Recover context across sessions

# Track file creation in beads
bd update [issue-id] --note "Created src/atoms/validators/email.js"
```

### Dependency Rules

**Critical enforcement rules that prevent architectural decay:**

**Atoms folder (`src/atoms/`):**
- ✅ NO imports from molecules, organisms, templates, or pages
- ✅ Only import from other atoms or external libraries
- ✅ Must be pure functions where possible
- ❌ NEVER import from `../molecules/` or `../organisms/`

**Molecules folder (`src/molecules/`):**
- ✅ ONLY import from `src/atoms/` and external libraries
- ❌ NEVER import from `../organisms/`, `../templates/`, or `../pages/`
- ❌ NEVER import from other molecules (avoid peer dependencies)

**Organisms folder (`src/organisms/`):**
- ✅ Import from `src/atoms/` and `src/molecules/`
- ❌ NEVER import from `../pages/` or other organisms at same level

**Templates folder (`src/templates/`):**
- ✅ Define contracts only (no implementation)
- ✅ Can reference other template definitions

**Pages folder (`src/pages/`):**
- ✅ Import from all levels (atoms, molecules, organisms, templates)
- ✅ Composition roots—wire everything together
- ❌ Nothing should import from pages (terminal nodes)

### Testing Strategy

Test files mirror source structure to maintain atomic design clarity:

```
tests/
  atoms/
    validators/
      email.test.js           # 1-3 focused unit tests
  molecules/
    validators/
      createUser.test.js      # 2-5 focused unit tests
  organisms/
    models/
      User.test.js            # 2-8 integration tests
```

**Alternative: Co-located tests**
```
src/
  atoms/
    validators/
      email.js
      email.test.js           # Co-located with implementation
```

Choose one pattern per project. Mirror structure preferred for larger codebases.

### Code Examples

#### Example 1: Atom Placement

```javascript
// src/atoms/validators/email.js
// Pure function, no dependencies on other project code
export function isValidEmail(email) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}
```

#### Example 2: Molecule Placement

```javascript
// src/molecules/validators/createUser.js
// Composes multiple atoms
import { isValidEmail } from '../../atoms/validators/email.js';
import { isValidPassword } from '../../atoms/validators/password.js';
import { sanitizeString } from '../../atoms/formatters/sanitize.js';

export function validateUserInput(userData) {
  return {
    email: isValidEmail(userData.email),
    password: isValidPassword(userData.password),
    username: sanitizeString(userData.username)
  };
}
```

#### Example 3: Organism Placement

```javascript
// src/organisms/models/User.js
// Complex composition with state and business logic
import { validateUserInput } from '../../molecules/validators/createUser.js';
import { hashPassword } from '../../molecules/helpers/crypto.js';
import { normalizeEmail } from '../../atoms/formatters/email.js';

export class User {
  constructor(userData) {
    this.data = userData;
  }

  async save() {
    const validation = validateUserInput(this.data);
    if (!validation.email) throw new Error('Invalid email');

    this.data.email = normalizeEmail(this.data.email);
    this.data.passwordHash = await hashPassword(this.data.password);
    // ... database logic
  }
}
```

#### Example 4: Import Rules Enforcement

```javascript
// ❌ WRONG: Atom importing from molecule
// src/atoms/validators/email.js
import { validateUserInput } from '../../molecules/validators/createUser.js'; // VIOLATES RULE

// ✅ CORRECT: Molecule importing from atoms
// src/molecules/validators/createUser.js
import { isValidEmail } from '../../atoms/validators/email.js'; // Valid downward dependency

// ❌ WRONG: Molecule importing from organism
// src/molecules/helpers/userHelper.js
import { User } from '../../organisms/models/User.js'; // VIOLATES RULE

// ✅ CORRECT: Organism importing from molecule
// src/organisms/models/User.js
import { validateUserInput } from '../../molecules/validators/createUser.js'; // Valid downward dependency
```

### Cross-References

- See `atomic-design.md` for complete atomic design principles
- See `coding-style.md` for naming and function size guidelines
- See `validation.md` for validation-specific organization patterns
