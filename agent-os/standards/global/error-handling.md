## Error handling best practices

- **User-Friendly Messages**: Provide clear, actionable error messages to users without exposing technical details or security information
- **Fail Fast and Explicitly**: Validate input and check preconditions early; fail with clear error messages rather than allowing invalid state
- **Specific Exception Types**: Use specific exception/error types rather than generic ones to enable targeted handling
- **Centralized Error Handling**: Handle errors at appropriate boundaries (controllers, API layers) rather than scattering try-catch blocks everywhere
- **Graceful Degradation**: Design systems to degrade gracefully when non-critical services fail rather than breaking entirely
- **Retry Strategies**: Implement exponential backoff for transient failures in external service calls
- **Clean Up Resources**: Always clean up resources (file handles, connections) in finally blocks or equivalent mechanisms

## Atomic Design Context

Error handling at each atomic level:
- **Atoms**: Return error values or throw exceptions. Pure error logic with no handling. Single responsibility: signal what went wrong.
- **Molecules**: Compose atom errors and add context. Propagate errors upward with additional information about the operation.
- **Organisms**: Catch and handle errors. Convert technical errors to user-friendly messages. Implement logging and recovery strategies.
- **Pages**: Global error boundaries, centralized logging, monitoring integrations. Handle unexpected errors gracefully.

## When to Use This Skill

- "I need to handle errors in [feature]"
- "Add error handling for [operation]"
- "Implement error recovery for [flow]"
- "Make [service] fail gracefully when [dependency] is down"
- "Add retry logic for [external API call]"

## BV/BD Integration Patterns

```bash
# Error handling typically implemented in organism beads
bv --recipe high-impact --filter "error-handling"

# Find error handling patterns in existing beads
bd --search "error boundary" --type organism

# Common bead structure for error handling organisms
beads/
  organisms/
    error-handler/
      error-handler.ts          # Main error handling logic
      error-logger.ts           # Logging integration
      error-boundary.tsx        # UI error boundary (if React)
```

## Dependency Rules

- **Atoms**: Throw or return errors. No dependencies on error handling infrastructure. Focus on validation and constraint checking.
- **Molecules**: Propagate errors with added context. May wrap atom errors in domain-specific error types.
- **Organisms**: Catch and handle errors. Depend on logging molecules, UI feedback atoms. Convert errors to user-facing messages.
- **Pages**: Global error boundaries only. Wire together organism error handlers. Integrate monitoring/observability tools.

## Testing Strategy

- **Atom tests**: Test error cases with invalid inputs. Verify correct error types are thrown/returned.
- **Molecule tests**: Test error propagation and context enrichment. Verify error information flows correctly.
- **Organism tests**: Test error handling behavior. Verify user feedback, logging calls, and recovery strategies.
- **Integration tests**: Test end-to-end error recovery flows. Verify graceful degradation and retry mechanisms.

## Code Examples

**Atom: Validator throws on invalid input**
```typescript
// atoms/validators/email-validator.ts
export function validateEmail(email: string): void {
  if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    throw new ValidationError('Invalid email format');
  }
}
```

**Molecule: Adds context to atom error**
```typescript
// molecules/user-validator.ts
export function validateUserInput(input: { email: string; age: number }) {
  try {
    validateEmail(input.email);
    validateAge(input.age);
  } catch (error) {
    throw new UserValidationError(`User validation failed: ${error.message}`, { input });
  }
}
```

**Organism: Catch, log, and provide user feedback**
```typescript
// organisms/user-registration-handler.ts
export async function handleUserRegistration(input: UserInput): Promise<Result> {
  try {
    validateUserInput(input);
    return await createUser(input);
  } catch (error) {
    logger.error('Registration failed', { error, input: sanitize(input) });

    if (error instanceof UserValidationError) {
      return { success: false, message: 'Please check your information and try again.' };
    }

    return { success: false, message: 'Registration temporarily unavailable. Please try again later.' };
  }
}
```

**Page: Global error boundary**
```tsx
// pages/app-error-boundary.tsx
export class AppErrorBoundary extends React.Component {
  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    monitoring.captureException(error, { context: errorInfo });
    logger.error('Unhandled React error', { error, errorInfo });
  }

  render() {
    if (this.state.hasError) {
      return <ErrorFallback message="Something went wrong. Please refresh the page." />;
    }
    return this.props.children;
  }
}
```
