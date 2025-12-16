## Security best practices

- **Defense in Depth**: Layer security controls at multiple levels (input validation, authentication, authorization, encryption)
- **Principle of Least Privilege**: Grant minimum necessary permissions; default to deny rather than allow
- **Secure by Default**: Make secure choices the default; require explicit action to disable security features
- **Input Validation**: Validate all user input at entry points; never trust client-side validation alone
- **Output Encoding**: Encode output based on context (HTML, JavaScript, SQL) to prevent injection attacks
- **Dependency Management**: Regularly audit and update dependencies; use tools to detect known vulnerabilities

## Atomic Design Context

Security at each atomic level:
- **Atoms**: Input validation and sanitization functions. Prevent XSS, SQL injection, command injection at the source. Pure functions with no side effects.
- **Molecules**: Authentication helpers (JWT validation, password hashing, token generation). Composable security primitives.
- **Organisms**: Authorization logic (role-based access control, permission checks), secure API endpoints with full auth/authz flow.
- **Pages**: CSRF protection, Content Security Policy headers, CORS configuration, secure session management. Global security policies.

## When to Use This Skill

- "Secure [endpoint/feature]" / "Add authentication to [service]"
- "Prevent [XSS/injection/CSRF]" / "Protect against [attack vector]"
- "Implement authorization for [resource]" / "Add RBAC to [API]"
- "Validate input for [form/endpoint]" / "Sanitize [user data]"
- "Add secure headers to [app]" / "Configure CSP for [site]"

## BV/BD Integration Patterns

```bash
# Security typically spans all atomic levels
bv --recipe high-impact --filter "security"

# Find security-critical beads requiring extra scrutiny
bd list --tag security-critical

# Common bead structure for security
beads/
  atoms/
    validators/              # Input validation, sanitization
  molecules/
    auth-helpers/            # JWT, password hashing, token generation
  organisms/
    authorization/           # RBAC, permission checking
    secure-endpoints/        # Protected API routes
  pages/
    security-middleware/     # CSP, CORS, CSRF protection
```

## Dependency Rules

- **Atoms**: Pure validation and sanitization. No dependencies on authentication or authorization. Focus on data integrity.
- **Molecules**: Authentication primitives (token validation, hashing). May depend on validation atoms. No authorization logic.
- **Organisms**: Full auth/authz implementation. Depend on auth molecules and validation atoms. Coordinate security at feature level.
- **Pages**: Global security policies (CSP, CORS, CSRF). Wire together organism security. No business logic.

## Testing Strategy

- **Security audits**: Run `npm audit`, Snyk, or OWASP Dependency-Check regularly on dependencies
- **Penetration testing**: Test critical authentication and authorization flows with attack simulations
- **Input fuzzing**: Fuzz validators with malicious inputs (XSS payloads, SQL injection attempts, path traversal)
- **Static analysis**: Use tools like ESLint security plugins, Semgrep for security patterns
- **Regular updates**: Keep dependencies current; automate security patch deployment

## Code Examples

**Atom: Input sanitization**
```typescript
// atoms/validators/sanitize-html.ts
import DOMPurify from 'isomorphic-dompurify';

export function sanitizeHtml(input: string): string {
  return DOMPurify.sanitize(input, {
    ALLOWED_TAGS: ['b', 'i', 'em', 'strong', 'a'],
    ALLOWED_ATTR: ['href']
  });
}

export function validateEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email) && email.length <= 254;
}
```

**Molecule: JWT validation**
```typescript
// molecules/auth/jwt-validator.ts
import jwt from 'jsonwebtoken';

export interface JwtPayload {
  userId: string;
  role: string;
  exp: number;
}

export function validateJwt(token: string, secret: string): JwtPayload | null {
  try {
    const payload = jwt.verify(token, secret) as JwtPayload;
    if (payload.exp < Date.now() / 1000) {
      return null; // Expired
    }
    return payload;
  } catch (error) {
    return null; // Invalid signature or malformed token
  }
}
```

**Organism: Protected API endpoint**
```typescript
// organisms/api/protected-endpoint.ts
import { validateJwt } from '../../molecules/auth/jwt-validator';
import { checkPermission } from '../../molecules/auth/permission-checker';

export async function protectedHandler(req: Request): Promise<Response> {
  const token = req.headers.get('Authorization')?.replace('Bearer ', '');

  if (!token) {
    return new Response('Unauthorized', { status: 401 });
  }

  const payload = validateJwt(token, process.env.JWT_SECRET!);
  if (!payload) {
    return new Response('Invalid or expired token', { status: 401 });
  }

  const hasPermission = await checkPermission(payload.userId, 'resource:read');
  if (!hasPermission) {
    return new Response('Forbidden', { status: 403 });
  }

  // Process authorized request
  return new Response(JSON.stringify({ data: 'protected content' }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' }
  });
}
```
