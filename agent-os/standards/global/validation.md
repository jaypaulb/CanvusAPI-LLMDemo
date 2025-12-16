## Atomic Design Context

| Level | Validation Responsibility | Examples |
|-------|--------------------------|----------|
| **Atom** | Single-field validators (pure, no dependencies) | `isEmail()`, `isPhone()`, `isRequired()`, `minLength(5)`, `maxLength(100)`, `isPositiveNumber()` |
| **Molecule** | Multi-field validators (compose atoms only) | `passwordConfirmation(pwd, confirm)`, `dateRange(start, end)`, `priceWithTax(price, tax)` |
| **Organism** | Entity validation (atoms + molecules + business logic) | `validateUserCreation(user)`, `validateOrder(order)`, `validatePayment(payment)` |
| **Page** | Form-level orchestration (run validators, collect/display errors) | Form submit handlers, API request validation, batch validation UI |

## When to Use This Skill

- "Validate [input/form/data]" → Apply appropriate atomic level
- "Add validation for [field/entity]" → Start with atoms, compose upward
- "Implement business rules for [feature]" → Organism-level validation
- User registration, checkout flows, data entry forms, API endpoints

## BV/BD Integration Patterns

```bash
# Find validation-related knowledge
bv --recipe actionable --filter "validation"

# Locate existing validators
bd list --tag validator

# Find validation utilities
bd search "validate" --type function
```

## Dependency Rules

- **Atom validators**: ZERO dependencies. Pure validation logic only. `(value) => boolean | error`
- **Molecule validators**: Compose ONLY atoms. NO business logic. `(field1, field2) => validationResult`
- **Organism validators**: Use molecules + atoms + business logic. May call services/repos for validation
- **Pages**: Orchestrate validation execution, aggregate errors, manage UI feedback. NO validation logic

## Testing Strategy

- **Atom tests**: Exhaustive edge cases (null, undefined, empty string, extreme values, malformed input)
- **Molecule tests**: Test valid/invalid combinations of multiple fields
- **Organism tests**: Test complete entity validation with realistic business scenarios
- **Page tests**: Integration tests verifying error display and form behavior

## Code Examples

### Email Validator (Atom)
```javascript
// Pure, single-field, no dependencies
export const isEmail = (value) => {
  const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return regex.test(value) || "Invalid email format";
};
```

### Password Confirmation (Molecule)
```javascript
// Composes atoms for multi-field validation
export const validatePasswordMatch = (password, confirmation) => {
  const minLength = 8;
  if (!password || password.length < minLength) return `Password must be at least ${minLength} characters`;
  if (password !== confirmation) return "Passwords do not match";
  return true;
};
```

### User Entity Validator (Organism)
```javascript
// Combines atoms, molecules, and business rules
export const validateUserCreation = (user, existingEmails = []) => {
  const errors = {};

  // Atom validations
  const emailResult = isEmail(user.email);
  if (emailResult !== true) errors.email = emailResult;

  // Molecule validations
  const pwdResult = validatePasswordMatch(user.password, user.passwordConfirm);
  if (pwdResult !== true) errors.password = pwdResult;

  // Business rule validation
  if (existingEmails.includes(user.email)) errors.email = "Email already registered";

  return Object.keys(errors).length === 0 ? { valid: true } : { valid: false, errors };
};
```

## Core Principles

- **Fail Early**: Validate at the earliest possible point before processing
- **Server-Side Required**: Never trust client-side validation alone
- **Specific Error Messages**: Tell users exactly what's wrong and how to fix it
- **Allowlists Over Blocklists**: Define what IS valid rather than blocking invalid patterns
- **Consistent Validation**: Same rules across all entry points (forms, APIs, imports)
