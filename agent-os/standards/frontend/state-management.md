# State Management Standards

## 1. Atomic Design Context

State management aligns with component hierarchy:

- **Atoms**: NO internal state (props only). Examples: `Button`, `Input`, `Icon`
- **Molecules**: Minimal local state (`useState` for toggles, input values). Examples: `SearchBar`, `ToggleSwitch`
- **Organisms**: Complex state (API data, hooks, side effects). Examples: `UserProfile`, `DataTable`
- **Pages**: Global state coordination (Redux, Context, Zustand providers). Examples: `DashboardPage`

## 2. When to Use This Skill

- "I need to manage state in [component]" / "Add state management for [feature]"
- "Implement [state pattern]" / "Create a custom hook for [data fetching]"

## 3. BV/BD Integration Patterns

```bash
bv --recipe actionable --filter "ui-organism" --filter "state"  # Organism beads with state
bv --recipe actionable --filter "api"  # API integration work
```

## 4. Dependency Rules

- **Atoms**: NEVER have internal state. Props only.
- **Molecules**: Local state ONLY (`useState`). NO API calls, NO global state, NO context.
- **Organisms**: Can use hooks, context, global store (read-only preferred), custom hooks, API calls.
- **Pages**: Global state providers (Redux, Context). Coordinate organism states.

## 5. Testing Strategy

- **Atom tests**: Props only (no state to test). Test callbacks invoked correctly.
- **Molecule tests**: Local state updates on user interaction. No mocking needed.
- **Organism tests**: State + side effects (API calls with mocking). Test loading/error states.
- **Integration tests**: Global state coordination. Test provider setup and state sharing.

## 6. Code Examples

### Example 1: Atom (Button - No State, Props Only)

```typescript
export const Button: React.FC<{ onClick?: () => void; children: ReactNode }> = ({ onClick, children }) => (
  <button onClick={onClick}>{children}</button>  // NO internal state, props only
);
```

### Example 2: Molecule (SearchBar - Local State)

```typescript
export const SearchBar: React.FC<{ onSearch: (q: string) => void }> = ({ onSearch }) => {
  const [query, setQuery] = useState(''); // Local UI state only
  return <div><Input value={query} onChange={(e) => setQuery(e.target.value)} /><Button onClick={() => onSearch(query)}>Search</Button></div>;
};
```

### Example 3: Organism (UserProfile - API State with Hooks)

```typescript
export const UserProfile: React.FC<{ userId: string }> = ({ userId }) => {
  const { user, updateUser, isLoading } = useUser(userId); // Custom hook for API state
  const [formData, setFormData] = useState({ name: '', email: '' });
  useEffect(() => { if (user) setFormData(user); }, [user]);

  return isLoading ? <div>Loading...</div> : <form><FormField label="Name" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} /><Button onClick={() => updateUser(formData)}>Save</Button></form>;
};
```

### Example 4: Global State Provider (Context/Redux)

```typescript
const AuthContext = createContext<{ user: User | null; login: (c: Credentials) => Promise<void> }>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  return <AuthContext.Provider value={{ user, login: async (c) => setUser(await api.login(c)) }}>{children}</AuthContext.Provider>;
};
```

## Core Best Practices

- **Minimal State**: Keep state local; lift only when multiple components need it
- **Single Source of Truth**: Each state piece has one canonical location
- **Immutable Updates**: Always create new objects/arrays, never mutate
- **Custom Hooks**: Extract complex state logic into reusable hooks
- **Loading States**: Always handle loading, error, and success states for async operations
