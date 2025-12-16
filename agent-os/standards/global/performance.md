## Performance Optimization Standard

### 1. Atomic Design Context

Performance optimization strategies map to atomic design levels:

- **Atoms**: Pure functions are inherently fast and predictable. Memoize expensive computational atoms. Profile before optimizing.
- **Molecules**: Composition overhead should be minimal. Avoid unnecessary object creation. Cache derived values from atoms.
- **Organisms**: Cache API responses, debounce user inputs, lazy load heavy components. Use React.memo strategically—not everywhere.
- **Pages**: Code splitting, route-based lazy loading, bundle optimization. Analyze bundle size regularly.

**Key Principle**: Optimize at the right level. Don't memoize cheap atoms. Do cache expensive organism operations.

### 2. When to Use This Skill

Trigger this standard when you encounter:

- "Optimize [component/function]" or "Improve performance of [feature]"
- "Reduce bundle size" or "Fix slow [operation]"
- Lighthouse scores below thresholds, slow render times, large bundle sizes
- User-facing latency issues or resource-intensive operations

### 3. BV/BD Integration Patterns

Use beads/bevies for performance work tracking:

```bash
# Find high-impact performance optimizations
bv --recipe high-impact --filter "performance"

# List existing optimization issues
bd list --tag optimization

# Create new performance issue
bd create "Optimize UserList rendering" -p 2 --tag optimization

# Link to related component issue
bd dep add [perf-issue-id] [component-issue-id] --type related
```

### 4. Dependency Rules

Performance work flows differently than feature work:

- **Atoms**: Optimize in isolation. Pure functions are easy to profile and benchmark independently.
- **Molecules**: Measure composition cost. Profile combined atom calls. Eliminate redundant computations.
- **Organisms**: Use React.memo, useMemo, useCallback strategically. Profile re-render frequency before optimizing.
- **Pages**: Analyze bundle size with webpack-bundle-analyzer. Lazy load routes. Split vendor bundles.

**Measurement First**: Always profile before optimizing. Premature optimization creates complexity without benefit.

### 5. Testing Strategy

Performance optimizations must be verified:

- **Performance budgets**: Set Lighthouse thresholds (e.g., FCP < 1.8s, TTI < 3.8s)
- **Bundle analysis**: Track bundle size over time (< 200KB initial load recommended)
- **Profiling tools**: Chrome DevTools Performance tab, React Profiler for component renders
- **Load testing**: For organism-level API endpoints, use k6 or similar tools
- **Benchmarking**: For atoms/molecules, use benchmark.js or similar for micro-benchmarks

**Regression prevention**: Add performance tests to CI for critical paths.

### 6. Code Examples

**Memoized Atom (Expensive Computation):**
```javascript
// atoms/fibonacci.js
const fibCache = new Map();

export function fibonacci(n) {
  if (n <= 1) return n;
  if (fibCache.has(n)) return fibCache.get(n);

  const result = fibonacci(n - 1) + fibonacci(n - 2);
  fibCache.set(n, result);
  return result;
}
```

**React.memo for Molecule Component:**
```javascript
// molecules/UserCard.js
import React from 'react';
import { formatDate } from '../atoms/formatDate';
import { Avatar } from '../atoms/Avatar';

const UserCard = React.memo(({ user, onSelect }) => (
  <div onClick={() => onSelect(user.id)}>
    <Avatar src={user.avatar} />
    <span>{user.name}</span>
    <span>{formatDate(user.lastSeen)}</span>
  </div>
), (prevProps, nextProps) => {
  // Custom comparison: only re-render if user data changed
  return prevProps.user.id === nextProps.user.id &&
         prevProps.user.lastSeen === nextProps.user.lastSeen;
});

export default UserCard;
```

**Lazy Loading Organism:**
```javascript
// pages/Dashboard.js
import React, { lazy, Suspense } from 'react';

const HeavyChart = lazy(() => import('../organisms/AnalyticsChart'));
const UserList = lazy(() => import('../organisms/UserList'));

export function Dashboard() {
  return (
    <div>
      <h1>Dashboard</h1>
      <Suspense fallback={<div>Loading chart...</div>}>
        <HeavyChart />
      </Suspense>
      <Suspense fallback={<div>Loading users...</div>}>
        <UserList />
      </Suspense>
    </div>
  );
}
```
