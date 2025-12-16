# UI Component Standards

## 1. Atomic Design Context

UI components exist at different levels of complexity in the atomic design hierarchy:

### Component Levels

- **Atoms**: Smallest indivisible UI units. No internal state, no composition of other UI components.
  - Examples: `Button`, `Input`, `Icon`, `Label`, `Badge`
  - Properties: Pure presentation, configurable via props, no side effects
  - Can use: Utility atoms (formatDate, formatCurrency), pure functions

- **Molecules**: Simple composition of 2-3 atoms with one clear purpose.
  - Examples: `SearchBar` (Input + Button + Icon), `FormField` (Label + Input + Error), `IconButton` (Icon + Button)
  - Properties: Simple interaction logic, minimal internal state
  - Can use: Only atoms from the level below

- **Organisms**: Complex composition of molecules and atoms with cohesive feature functionality.
  - Examples: `UserProfile` (Avatar + FormFields + IconButtons + state), `DataTable` (complex composition with sorting/filtering), `LoginForm` (multiple FormFields + validation + submission)
  - Properties: Feature-level logic, state management, API integration via hooks/services
  - Can use: Molecules, atoms, external services/hooks

### Composition Hierarchy Examples

```
Button (atom) → IconButton (molecule) → Toolbar (organism)
Input (atom) → FormField (molecule) → LoginForm (organism)
Icon (atom) → SearchBar (molecule) → Dashboard (organism)
```

## 2. When to Use This Skill

Claude Code should apply this skill when encountering:

- "I need to create a reusable UI component for [element]"
- "I'm building a [component type] component"
- "I need to add a form for [entity]"
- "I'm implementing the [feature] UI"
- "Create a [Button/Input/Card/Modal] component"
- "Build a navigation bar for the application"
- "Add a data visualization component for [data]"

## 3. BV/BD Integration Patterns

When working in beads mode, follow these patterns for UI component development:

### Check for Unblocked UI Work

```bash
# Find UI component work that's ready to implement
bv --recipe actionable --format json | jq '.[] | select(.tags[]? == "ui-organism")'

# Find foundational UI atoms that unblock dependent work
bv --robot-insights --format json | jq '.influencers[] | select(.tags[]? == "ui-atom")'
```

### Bottom-Up Development Flow

```bash
# 1. Start with atoms
bv --recipe actionable --filter "ui-atom" --format json

# 2. After completing atoms, verify molecules can proceed
bd show bd-501 --rdeps  # Show what depends on this atom

# 3. Then build molecules
bv --recipe actionable --filter "ui-molecule" --format json

# 4. Finally build organisms
bv --recipe actionable --filter "ui-organism" --format json
```

### Tagging Strategy

When creating beads for UI components:
- Tag atoms: `ui-atom`, specific component name (e.g., `button`, `input`)
- Tag molecules: `ui-molecule`, feature area (e.g., `search`, `forms`)
- Tag organisms: `ui-organism`, feature name (e.g., `user-profile`, `data-table`)

## 4. Dependency Rules

Strict rules for component dependencies to maintain clean atomic hierarchy:

### Atoms
- **ONLY** depend on: Utility atoms, pure functions, constants
- **NEVER** depend on: Other UI atoms, molecules, organisms, API calls

### Molecules
- **ONLY** depend on: 2-3 specific atoms (not just "any atoms")
- **NEVER** depend on: Other molecules, organisms, API calls directly

### Organisms
- **CAN** depend on: Molecules, atoms, API calls via hooks/services, state management
- **NEVER** depend on: Other organisms (use composition instead)

### Violation Examples (DO NOT DO THIS)

```typescript
// ❌ Atom depending on another atom
const Button = () => {
  return <IconButton />; // Button is atom, IconButton is molecule
};

// ❌ Molecule depending on organism
const SearchBar = () => {
  return <UserProfile />; // Mixing levels
};

// ❌ Organism depending on organism
const Dashboard = () => {
  return <UserProfile />; // Should compose shared molecules instead
};
```

## 5. Testing Strategy

Different atomic levels require different testing approaches:

### Atoms: Unit Tests
- Render correctness with different props
- Accessibility (ARIA labels, roles)
- Keyboard navigation
- Visual variants (sizes, colors, states)

```typescript
describe('Button', () => {
  it('renders with correct text', () => { /* ... */ });
  it('calls onClick when clicked', () => { /* ... */ });
  it('is accessible via keyboard', () => { /* ... */ });
  it('renders disabled state correctly', () => { /* ... */ });
});
```

### Molecules: Unit Tests (Composition + Interaction)
- Component composition is correct
- Interaction between atoms works
- Internal state updates properly
- Props are passed to atoms correctly

```typescript
describe('SearchBar', () => {
  it('renders input and button', () => { /* ... */ });
  it('updates input value on change', () => { /* ... */ });
  it('calls onSearch when button clicked', () => { /* ... */ });
  it('clears input when clear icon clicked', () => { /* ... */ });
});
```

### Organisms: Integration Tests
- State management works end-to-end
- API calls are made correctly (with mocking)
- Side effects execute properly
- Complex user flows complete

```typescript
describe('LoginForm', () => {
  it('submits form with valid data', async () => { /* mock API */ });
  it('displays validation errors', () => { /* ... */ });
  it('handles API errors gracefully', () => { /* ... */ });
  it('redirects after successful login', () => { /* ... */ });
});
```

### Visual Regression & Accessibility
- **Snapshot tests**: All atoms, selected molecules
- **Accessibility tests**: ARIA labels, screen reader support, keyboard navigation
- **Visual regression**: Use tools like Chromatic or Percy for atoms and molecules

## 6. Code Examples

### Example 1: Atom Component (Button)

```typescript
// src/components/atoms/Button.tsx
interface ButtonProps {
  variant?: 'primary' | 'secondary' | 'danger';
  size?: 'small' | 'medium' | 'large';
  disabled?: boolean;
  children: React.ReactNode;
  onClick?: () => void;
}

export const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  size = 'medium',
  disabled = false,
  children,
  onClick,
}) => {
  return (
    <button
      className={`btn btn-${variant} btn-${size}`}
      disabled={disabled}
      onClick={onClick}
      aria-disabled={disabled}
    >
      {children}
    </button>
  );
};

// NO dependencies on other UI components
// NO state management
// NO API calls
```

### Example 2: Molecule Component (SearchBar)

```typescript
// src/components/molecules/SearchBar.tsx
import { Input } from '../atoms/Input';
import { Button } from '../atoms/Button';
import { Icon } from '../atoms/Icon';

interface SearchBarProps {
  placeholder?: string;
  onSearch: (query: string) => void;
}

export const SearchBar: React.FC<SearchBarProps> = ({
  placeholder = 'Search...',
  onSearch,
}) => {
  const [query, setQuery] = React.useState('');

  const handleSearch = () => {
    onSearch(query);
  };

  return (
    <div className="search-bar">
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={placeholder}
      />
      <Button onClick={handleSearch}>
        <Icon name="search" />
      </Button>
    </div>
  );
};

// Depends on: Input, Button, Icon (all atoms)
// Simple internal state (query)
// NO API calls
```

### Example 3: Organism Component (UserProfile)

```typescript
// src/components/organisms/UserProfile.tsx
import { FormField } from '../molecules/FormField';
import { IconButton } from '../molecules/IconButton';
import { Avatar } from '../atoms/Avatar';
import { useUpdateUser } from '../../hooks/useUpdateUser';

interface UserProfileProps {
  userId: string;
}

export const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
  const { user, updateUser, isLoading } = useUpdateUser(userId);
  const [formData, setFormData] = React.useState({
    name: user?.name || '',
    email: user?.email || '',
  });

  const handleSubmit = async () => {
    await updateUser(formData);
  };

  return (
    <div className="user-profile">
      <Avatar src={user?.avatar} size="large" />

      <FormField
        label="Name"
        value={formData.name}
        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
      />

      <FormField
        label="Email"
        type="email"
        value={formData.email}
        onChange={(e) => setFormData({ ...formData, email: e.target.value })}
      />

      <IconButton
        icon="save"
        onClick={handleSubmit}
        disabled={isLoading}
      >
        Save Changes
      </IconButton>
    </div>
  );
};

// Depends on: FormField (molecule), IconButton (molecule), Avatar (atom)
// Complex state management
// API integration via hooks
```

### Example 4: Using Utility Atoms in Components

```typescript
// src/components/atoms/Timestamp.tsx
import { formatDate } from '../../utils/atoms/formatDate'; // utility atom

interface TimestampProps {
  date: Date;
  format?: 'short' | 'long';
}

export const Timestamp: React.FC<TimestampProps> = ({ date, format = 'short' }) => {
  return (
    <time dateTime={date.toISOString()}>
      {formatDate(date, format)}
    </time>
  );
};

// Atom can use utility atoms (pure functions)
// NO dependencies on other UI components
```

### Example 5: Atomic Component Composition Hierarchy

```typescript
// Atom level
const Button = ({ children, onClick }) => <button onClick={onClick}>{children}</button>;
const Input = ({ value, onChange }) => <input value={value} onChange={onChange} />;
const Label = ({ children }) => <label>{children}</label>;
const ErrorText = ({ children }) => <span className="error">{children}</span>;

// Molecule level (combines atoms)
const FormField = ({ label, value, onChange, error }) => (
  <div>
    <Label>{label}</Label>
    <Input value={value} onChange={onChange} />
    {error && <ErrorText>{error}</ErrorText>}
  </div>
);

// Organism level (combines molecules + state + logic)
const LoginForm = ({ onSubmit }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState({});

  return (
    <form onSubmit={onSubmit}>
      <FormField
        label="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        error={errors.email}
      />
      <FormField
        label="Password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        error={errors.password}
      />
      <Button onClick={handleSubmit}>Login</Button>
    </form>
  );
};
```

## Core Best Practices

- **Single Responsibility**: Each component should have one clear purpose and do it well
- **Reusability**: Design components to be reused across different contexts with configurable props
- **Composability**: Build complex UIs by combining smaller, simpler components rather than monolithic structures
- **Clear Interface**: Define explicit, well-documented props with sensible defaults for ease of use
- **Encapsulation**: Keep internal implementation details private and expose only necessary APIs
- **Consistent Naming**: Use clear, descriptive names that indicate the component's purpose and follow team conventions
- **State Management**: Keep state as local as possible; lift it up only when needed by multiple components
- **Minimal Props**: Keep the number of props manageable; if a component needs many props, consider composition or splitting it
- **Documentation**: Document component usage, props, and provide examples for easier adoption by team members
