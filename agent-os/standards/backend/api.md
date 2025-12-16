## API endpoint standards and conventions

## 1. Atomic Design Context

API endpoints are **ORGANISMS** in the atomic design hierarchy. They compose multiple smaller units to create cohesive, feature-complete units.

**API Layer Position:**
- **Level**: Organism (API layer)
- **Depends on**: Models (database organisms), molecules (request validators, response formatters), atoms (string utilities, constants)
- **Never depends on**: UI organisms, frontend components, other API endpoints for shared logic
- **Composes into**: Pages (application entry points, route definitions)

**Concrete Examples:**
- `POST /users` endpoint (organism) uses:
  - `User` model (database organism)
  - `emailValidator` (atom - pure function)
  - `hashPassword` (atom - pure function)
  - `createUserDTO` (molecule - composite validator)
  - `JWT` generator (molecule - token creation + signing)
- `GET /products?page=1&limit=20` endpoint (organism) uses:
  - `Product` model (database organism)
  - `paginationParams` (molecule - query parser + validator)
  - `formatProductResponse` (molecule - data transformer)
  - `HTTP_STATUS` constants (atom)

**Key Principle**: API endpoints orchestrate business logic but don't contain it. Extract reusable logic to molecules and atoms.

## 2. When to Use This Skill

Claude Code should apply API design standards when encountering these phrases:

- "I need to create a REST API endpoint for [resource]"
- "I'm adding authentication to an endpoint"
- "I need to implement [HTTP method] for [entity]"
- "I'm building a controller for [feature]"
- "Create an API route that handles [action]"
- "Add validation to the [endpoint] request"
- "Implement error handling for [API route]"
- "Set up pagination for [resource] endpoint"
- "Add rate limiting to [API]"
- "Create middleware for [cross-cutting concern]"

## 3. Core Design Standards

- **RESTful Design**: Follow REST principles with clear resource-based URLs and appropriate HTTP methods (GET, POST, PUT, PATCH, DELETE)
- **Consistent Naming**: Use consistent, lowercase, hyphenated or underscored naming conventions for endpoints across the API
- **Versioning**: Implement API versioning strategy (URL path or headers) to manage breaking changes without disrupting existing clients
- **Plural Nouns**: Use plural nouns for resource endpoints (e.g., `/users`, `/products`) for consistency
- **Nested Resources**: Limit nesting depth to 2-3 levels maximum to keep URLs readable and maintainable
- **Query Parameters**: Use query parameters for filtering, sorting, pagination, and search rather than creating separate endpoints
- **HTTP Status Codes**: Return appropriate, consistent HTTP status codes that accurately reflect the response (200, 201, 400, 404, 500, etc.)
- **Rate Limiting Headers**: Include rate limit information in response headers to help clients manage their usage

## 4. BV/BD Integration Patterns

When working in beads mode, use these patterns to discover and track API work:

```bash
# Check if API work is high-impact
bv --recipe high-impact --format json | jq '.[] | select(.tags[]? == "api-organism")'

# Find ready API work (all dependencies satisfied)
bv --recipe actionable --format json | jq '.[] | select(.tags[]? == "api-organism")'

# After implementation, verify integration dependencies
bd show bd-401 --deps  # Show what this API organism depends on

# Check what depends on this API (impact analysis)
bd show bd-401 --dependents  # What breaks if this changes?
```

**Tagging Strategy:**
- Tag API endpoint beads with: `api-organism`, `backend`, `[resource-name]`
- Tag validator molecules with: `validator-molecule`, `api-dependency`
- Tag model dependencies with: `model-organism`, `database`

## 5. Dependency Rules

API layer organisms must follow strict dependency rules:

**CAN depend on:**
- Models (database organisms) - for data persistence
- Molecules (validators, formatters, parsers) - for reusable composition
- Atoms (string utilities, constants, pure functions) - for primitives
- Service layer organisms - for shared business logic
- Middleware molecules - for request/response processing

**CANNOT depend on:**
- UI organisms - breaks separation of concerns
- Other API endpoints - creates coupling; use service layer instead
- Frontend components - wrong layer
- Direct database connections in route handlers - use models/repositories

**Test dependencies:**
- `test-writer-organism` - for integration tests
- `mock-generator-molecule` - for test data
- `test-utilities-atom` - for assertions and helpers

## 6. Testing Strategy

Follow the test pyramid: more unit tests than integration, more integration than E2E.

**Unit Tests** (60% of coverage):
- Controller logic isolated from database
- Request validation with various inputs
- Response formatting with edge cases
- Error handling for all failure modes

**Integration Tests** (30% of coverage):
- Full request/response cycle
- Database interactions (with test DB)
- Authentication/authorization flows
- Middleware execution order

**E2E Tests** (10% of coverage):
- Complete user workflows
- Multi-endpoint transactions
- Real authentication flows
- Error recovery scenarios

## 7. Code Examples

### Example 1: REST Endpoint Using Model + Validator Atom

```javascript
// atoms/validators/email.js
export const isValidEmail = (email) => {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
};

// molecules/validators/createUser.js
import { isValidEmail } from '../../atoms/validators/email.js';

export const validateCreateUser = (data) => {
  const errors = {};
  if (!data.email || !isValidEmail(data.email)) {
    errors.email = 'Invalid email address';
  }
  if (!data.password || data.password.length < 8) {
    errors.password = 'Password must be at least 8 characters';
  }
  return { isValid: Object.keys(errors).length === 0, errors };
};

// organisms/controllers/userController.js
import { User } from '../models/User.js';
import { validateCreateUser } from '../../molecules/validators/createUser.js';

export const createUser = async (req, res) => {
  const { isValid, errors } = validateCreateUser(req.body);
  if (!isValid) {
    return res.status(400).json({ errors });
  }

  const user = await User.create(req.body);
  res.status(201).json({ data: user });
};
```

### Example 2: Authentication Middleware Using JWT Molecule

```javascript
// molecules/auth/jwt.js
import jwt from 'jsonwebtoken';

export const generateToken = (payload) => {
  return jwt.sign(payload, process.env.JWT_SECRET, { expiresIn: '7d' });
};

export const verifyToken = (token) => {
  try {
    return { valid: true, payload: jwt.verify(token, process.env.JWT_SECRET) };
  } catch (err) {
    return { valid: false, error: err.message };
  }
};

// organisms/middleware/authenticate.js
import { verifyToken } from '../../molecules/auth/jwt.js';

export const authenticate = (req, res, next) => {
  const token = req.headers.authorization?.replace('Bearer ', '');
  if (!token) {
    return res.status(401).json({ error: 'No token provided' });
  }

  const { valid, payload, error } = verifyToken(token);
  if (!valid) {
    return res.status(401).json({ error });
  }

  req.user = payload;
  next();
};
```

### Example 3: Request Validation Using Composite Validator Molecule

```javascript
// molecules/validators/pagination.js
export const validatePaginationParams = (query) => {
  const page = Math.max(1, parseInt(query.page) || 1);
  const limit = Math.min(100, Math.max(1, parseInt(query.limit) || 20));
  return { page, limit, offset: (page - 1) * limit };
};

// organisms/controllers/productController.js
import { Product } from '../models/Product.js';
import { validatePaginationParams } from '../../molecules/validators/pagination.js';

export const listProducts = async (req, res) => {
  const { page, limit, offset } = validatePaginationParams(req.query);

  const [products, total] = await Promise.all([
    Product.findAll({ limit, offset }),
    Product.count()
  ]);

  res.status(200).json({
    data: products,
    meta: { page, limit, total, totalPages: Math.ceil(total / limit) }
  });
};
```

### Example 4: Error Handling with Standardized Response Format

```javascript
// atoms/constants/httpStatus.js
export const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  INTERNAL_ERROR: 500
};

// atoms/constants/errorMessages.js
export const ERROR_MESSAGES = {
  NOT_FOUND: 'Resource not found',
  UNAUTHORIZED: 'Authentication required',
  VALIDATION_FAILED: 'Validation failed'
};

// molecules/formatters/errorResponse.js
import { ERROR_MESSAGES } from '../../atoms/constants/errorMessages.js';

export const formatErrorResponse = (type, details = {}) => {
  return {
    error: {
      message: ERROR_MESSAGES[type] || 'An error occurred',
      ...details
    }
  };
};

// organisms/middleware/errorHandler.js
import { HTTP_STATUS } from '../../atoms/constants/httpStatus.js';
import { formatErrorResponse } from '../../molecules/formatters/errorResponse.js';

export const errorHandler = (err, req, res, next) => {
  console.error(err);

  if (err.name === 'ValidationError') {
    return res.status(HTTP_STATUS.BAD_REQUEST)
      .json(formatErrorResponse('VALIDATION_FAILED', { fields: err.fields }));
  }

  res.status(HTTP_STATUS.INTERNAL_ERROR)
    .json(formatErrorResponse('INTERNAL_ERROR'));
};
```

### Example 5: Pagination Helper Molecule

```javascript
// molecules/helpers/pagination.js
export const createPaginationLinks = (baseUrl, page, limit, total) => {
  const totalPages = Math.ceil(total / limit);
  const links = {
    self: `${baseUrl}?page=${page}&limit=${limit}`,
    first: `${baseUrl}?page=1&limit=${limit}`,
    last: `${baseUrl}?page=${totalPages}&limit=${limit}`
  };

  if (page > 1) {
    links.prev = `${baseUrl}?page=${page - 1}&limit=${limit}`;
  }
  if (page < totalPages) {
    links.next = `${baseUrl}?page=${page + 1}&limit=${limit}`;
  }

  return links;
};

// organisms/controllers/orderController.js
import { Order } from '../models/Order.js';
import { validatePaginationParams } from '../../molecules/validators/pagination.js';
import { createPaginationLinks } from '../../molecules/helpers/pagination.js';

export const listOrders = async (req, res) => {
  const { page, limit, offset } = validatePaginationParams(req.query);

  const [orders, total] = await Promise.all([
    Order.findAll({ limit, offset }),
    Order.count()
  ]);

  res.status(200).json({
    data: orders,
    meta: { page, limit, total },
    links: createPaginationLinks('/api/orders', page, limit, total)
  });
};
