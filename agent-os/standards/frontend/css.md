# CSS Standards

## 1. Atomic Design Context

CSS aligns with atomic component hierarchy:

- **Atom styles**: Base elements (button, input, icon) - minimal, reusable, no dependencies
- **Molecule styles**: Composed components (search-bar, form-field) - composition-specific
- **Organism styles**: Complex layouts (user-profile, data-table) - feature-specific
- Avoid cross-component dependencies in styles - each level only depends on levels below

## 2. When to Use This Skill

- "I need to style the [component]"
- "Add CSS for [feature]"
- "Create styles for [UI element]"
- "Make this responsive/accessible"

## 3. BV/BD Integration Patterns

```bash
# CSS work typically aligns with UI component beads
bd show bd-501 --deps  # Check if atom CSS is complete before molecule CSS
bd new "Style search-bar molecule" --depends bd-450  # Depends on button atom
```

## 4. Dependency Rules

- **Atom CSS**: Only global variables/tokens (colors, spacing, typography)
- **Molecule CSS**: Only atom CSS classes - compose, don't override
- **Organism CSS**: Atom + molecule CSS classes - layout and integration only
- **Never**: Sideways dependencies (molecule → molecule, organism → organism)

## 5. Core Practices

- **Consistent Methodology**: Apply and stick to the project's consistent CSS methodology (Tailwind, BEM, utility classes, CSS modules, etc.) across the entire project
- **Avoid Overriding Framework Styles**: Work with your framework's patterns rather than fighting against them with excessive overrides
- **Maintain Design System**: Establish and document design tokens (colors, spacing, typography) for consistency
- **Minimize Custom CSS**: Leverage framework utilities and components to reduce custom CSS maintenance burden
- **Performance Considerations**: Optimize for production with CSS purging/tree-shaking to remove unused styles

## 6. Testing Strategy

- **Atoms**: Visual regression tests (snapshot), all variants tested
- **Molecules**: Responsive design tests, composition integrity
- **Organisms**: End-to-end layout tests, cross-browser checks
- **Always**: Accessibility checks (contrast ratios ≥4.5:1, keyboard focus visible)

## 7. Code Examples

**Atom CSS (button with variants):**
```css
/* Base atom - single responsibility */
.btn {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  font-weight: 600;
}
.btn-primary { background: var(--color-primary); color: white; }
.btn-secondary { background: var(--color-gray-200); color: var(--color-gray-800); }
```

**Molecule CSS (search-bar composition):**
```css
/* Compose atoms, don't redefine */
.search-bar {
  display: flex;
  gap: var(--space-2);
}
.search-bar .btn { flex-shrink: 0; } /* Only layout concerns */
```

**CSS Variables (design tokens):**
```css
:root {
  --color-primary: #3b82f6;
  --space-2: 0.5rem;
  --space-4: 1rem;
  --radius-md: 0.375rem;
}
```
