## Documentation Standard

### 1. Atomic Design Context

Documentation granularity mirrors the atomic design hierarchy:

- **Atoms**: Inline JSDoc/docstrings (function signatures, parameter descriptions, return types)
- **Molecules**: README per molecule explaining composition and purpose
- **Organisms**: Architecture docs (how it integrates atoms/molecules, design decisions)
- **Pages**: API documentation, user guides, integration examples

### 2. When to Use This Skill

Invoke documentation work when:
- "Document [API/function/component]"
- "Add docs for [feature]"
- "Write README for [module]"
- "Generate API docs"

### 3. BV/BD Integration Patterns

Track documentation work through beads:

```bash
# Find documentation tasks
bd list --tag documentation

# Find undocumented code
bv --recipe stale --filter "undocumented"

# Create documentation issue
bd create "Document auth molecule" -p 2 --tag documentation
```

### 4. Dependency Rules

- **Co-location**: Documentation lives alongside code (`atoms/validators/email.js` + `email.md`)
- **Generated docs**: API docs auto-generated from code (JSDoc, TSDoc, docstrings)
- **User docs**: Separate in `/docs` folder (guides, tutorials, architecture)
- **READMEs**: Per-directory for molecules/organisms explaining structure

### 5. Testing Strategy

- **Doc tests**: Verify code examples compile and run
- **Link checking**: Ensure internal/external links valid
- **Freshness checks**: Docs updated with code changes (CI check)
- **Completeness**: Public APIs must have docs (linter enforced)

### 6. Code Examples

**Atom with JSDoc:**
```javascript
/**
 * Validates email format according to RFC 5322.
 * @param {string} email - The email address to validate
 * @returns {boolean} True if valid, false otherwise
 * @example
 * validateEmail('user@example.com') // => true
 * validateEmail('invalid') // => false
 */
export function validateEmail(email) {
  const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return regex.test(email);
}
```

**Molecule with README:**
```markdown
# User Validation Molecule

Composes email and password validation atoms.

## Atoms Used
- `validateEmail` from `atoms/validators/email.js`
- `checkPasswordStrength` from `atoms/validators/password.js`

## Purpose
Validates user registration input before database write.

## Usage
\`\`\`javascript
import { validateUserInput } from './molecules/user-validator.js';

const result = validateUserInput({ email: 'user@example.com', password: 'SecureP@ss123' });
// => { valid: true, errors: [] }
\`\`\`
```
