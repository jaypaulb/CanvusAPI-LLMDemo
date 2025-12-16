# UI Accessibility Standards

## 1. Atomic Design Context

Accessibility requirements scale with component complexity:

- **Atoms**: Built-in accessibility through semantic HTML and ARIA primitives. Examples: `Button` (role, aria-label, keyboard support), `Input` (label association, aria-describedby)
- **Molecules**: Composite ARIA relationships and announcements. Examples: `SearchBar` (aria-label for combined purpose), `FormField` (label + input + error announcements)
- **Organisms**: Focus management and screen reader context. Examples: `Modal` (focus trapping, aria-modal), `DataTable` (aria-sort, row/column headers)
- **Pages**: Document structure and landmark regions. Examples: `DashboardPage` (skip links, main/nav/aside landmarks, h1-h6 hierarchy)

## 2. When to Use This Skill

- "Make [component] accessible" / "Add ARIA to [element]"
- "Ensure [feature] works with keyboard/screen reader"
- "Fix accessibility issues in [component]"
- "Add keyboard navigation to [feature]"
- "Improve screen reader support for [element]"

## 3. BV/BD Integration Patterns

```bash
# Find accessibility work ready to implement
bv --recipe actionable --filter "accessibility"
bv --recipe actionable --filter "a11y"

# Find accessibility beads by tag
bd list --tag a11y
bd list --tag wcag

# Check for accessibility blockers
bv --robot-insights --format json | jq '.blockers[] | select(.tags[]? == "a11y")'
```

## 4. Dependency Rules

- **Atoms**: Built-in accessibility ONLY (semantic HTML, native ARIA). NO custom focus management, NO announcements.
- **Molecules**: Compose accessible atoms. Add composite ARIA (aria-labelledby spanning atoms). Minimal announcements.
- **Organisms**: Manage focus (modals, dialogs). Announce dynamic changes (live regions). Handle complex interactions.
- **Pages**: Document structure (landmarks, headings). Skip links. Focus restoration on navigation.

## 5. Testing Strategy

### Automated Testing
- **axe-core**: Integrate into unit/integration tests to catch ARIA/role violations
- **WAVE**: Browser extension for visual accessibility review
- **Lighthouse**: CI/CD accessibility audits (minimum score: 90)

### Manual Testing
- **Keyboard navigation**: Tab order, Enter/Space activation, Escape dismissal, arrow navigation
- **Screen readers**: NVDA (Windows), JAWS (Windows), VoiceOver (macOS/iOS)
- **WCAG 2.1 AA compliance**: Minimum standard for all components

## 6. Code Examples

### Example 1: Accessible Button Atom

```typescript
export const Button: React.FC<{ onClick?: () => void; children: ReactNode; variant?: string; ariaLabel?: string }> = ({ onClick, children, variant = 'primary', ariaLabel }) => (
  <button
    onClick={onClick}
    className={`btn-${variant}`}
    aria-label={ariaLabel || undefined}  // Override accessible name if needed
    type="button"  // Prevent accidental form submission
  >
    {children}
  </button>
);
// Semantic HTML ensures keyboard support (Tab, Enter/Space)
// ARIA label for icon-only buttons
```

### Example 2: Accessible FormField Molecule

```typescript
export const FormField: React.FC<{ label: string; value: string; onChange: ChangeEventHandler; error?: string }> = ({ label, value, onChange, error }) => {
  const id = useId();  // React 18+ unique ID
  const errorId = `${id}-error`;

  return (
    <div>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        value={value}
        onChange={onChange}
        aria-invalid={!!error}
        aria-describedby={error ? errorId : undefined}  // Associate error with input
      />
      {error && <ErrorText id={errorId} role="alert">{error}</ErrorText>}  {/* Live announcement */}
    </div>
  );
};
```

### Example 3: Accessible Modal Organism (Focus Management)

```typescript
export const Modal: React.FC<{ isOpen: boolean; onClose: () => void; children: ReactNode }> = ({ isOpen, onClose, children }) => {
  const modalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    const previousFocus = document.activeElement as HTMLElement;
    modalRef.current?.focus();  // Move focus into modal

    return () => previousFocus?.focus();  // Restore focus on close
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div role="dialog" aria-modal="true" ref={modalRef} tabIndex={-1} onKeyDown={(e) => e.key === 'Escape' && onClose()}>
      {children}
    </div>
  );
};
```

## Core Best Practices

- **Semantic HTML First**: Use `<button>`, `<nav>`, `<main>`, `<aside>` over generic `<div>` with ARIA
- **Keyboard Navigation**: Ensure all interactive elements are reachable and operable via keyboard alone
- **Color Contrast**: Maintain 4.5:1 for normal text, 3:1 for large text (WCAG AA)
- **Focus Indicators**: Visible focus states for all interactive elements (no `outline: none` without replacement)
- **Alternative Text**: Meaningful alt text for images; empty alt for decorative images
- **ARIA When Needed**: Use ARIA to enhance, not replace, semantic HTML (first rule of ARIA: don't use ARIA)
- **Screen Reader Testing**: Test with actual assistive technology, not just automated tools
