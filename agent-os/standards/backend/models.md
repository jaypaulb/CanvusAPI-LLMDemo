# Database Models Standard

## Atomic Design Context

Database models are **ORGANISMS** in the atomic design hierarchy. They represent the database layer of your application and should be composed from smaller, reusable units.

**Dependency Rules:**
- **CAN depend on:**
  - Atoms: validators (email, phone), formatters (currency, date), constants (enum values, defaults)
  - Molecules: composite validators (password strength), field helpers (hash + salt), domain validators (business rules)
- **CANNOT depend on:**
  - Other organisms: API handlers, controllers, UI components, service layers
  - Other models directly in code (use associations/foreign keys instead)
  - Test organisms (test-writer should depend on models, not vice versa)

**Architecture Position:**
```
Pages/Templates (API endpoints, app bootstrap)
    ↓
Organisms (models, services, controllers) ← YOU ARE HERE
    ↓
Molecules (field helpers, composite validators)
    ↓
Atoms (validators, formatters, constants)
```

**Examples:**
- User model uses email validator atom
- User model uses password hasher molecule (hash + salt + pepper)
- Product model uses currency formatter atom and price validator molecule
- Order model uses status enum atom and state machine molecule

## When to Use This Skill

Claude Code should apply database model patterns when encountering:

- "I need to create a database model for [entity]"
- "I'm adding a new table to the schema"
- "I need to define model associations/relationships"
- "I'm setting up migrations for database changes"
- "I need to add validations to the [entity] model"
- "I'm implementing soft deletes for [entity]"
- "I need to add scopes/queries to the [entity] model"
- "I'm creating a join table for many-to-many relationships"

## Database Model Best Practices

- **Clear Naming**: Use singular names for models and plural for tables following your framework's conventions
- **Timestamps**: Include created and updated timestamps on all tables for auditing and debugging
- **Data Integrity**: Use database constraints (NOT NULL, UNIQUE, foreign keys) to enforce data rules at the database level
- **Appropriate Data Types**: Choose data types that match the data's purpose and size requirements
- **Indexes on Foreign Keys**: Index foreign key columns and other frequently queried fields for performance
- **Validation at Multiple Layers**: Implement validation at both model and database levels for defense in depth
- **Relationship Clarity**: Define relationships clearly with appropriate cascade behaviors and naming conventions
- **Avoid Over-Normalization**: Balance normalization with practical query performance needs

## BV/BD Integration Patterns

When working in beads mode, integrate models into your business value flow:

```bash
# Check if this model is on the critical path for value delivery
bv --robot-insights --format json | jq '.keystones[] | select(.type == "database-model")'

# Before implementing, check which beads depend on this model
bd show bd-301 --deps

# After implementation, verify no circular dependencies introduced
bv --robot-insights --format json | jq '.cycles'

# Update bead status after model + migration complete
bd done bd-301 "User model with email validation and password hashing"
```

**Integration Checklist:**
- Model completion should mark corresponding database bead as done
- Migrations should be created atomically with model changes
- Test beads should depend on model beads (not vice versa)
- API beads should depend on model beads (enforces layering)

## Testing Strategy

**Unit Tests (Model Logic):**
- Model methods and instance behavior
- Validations fire correctly
- Scopes return expected results
- Default values are set
- Computed properties work correctly

**Integration Tests (Database Interaction):**
- Associations load correctly (belongs_to, has_many, has_one)
- Cascade behaviors work (delete, nullify)
- Transactions roll back properly
- Unique constraints are enforced
- Foreign key constraints prevent orphans

**Migration Tests:**
- Test both `up` AND `down` migrations
- Verify indexes are created
- Confirm constraints are enforced
- Check default values are set
- Validate data type conversions

**Fixture Strategy:**
- Atomic fixtures: Single model instance, minimal data
- Molecular fixtures: Related models (user + posts), realistic data
- Use factories/builders for test data generation
- Avoid fixture interdependencies that create ordering issues

## Code Examples

### Example 1: Model Using Atom Validator

```python
# atoms/validators.py
def is_valid_email(email: str) -> bool:
    """Atom: Pure email validation logic"""
    import re
    pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    return bool(re.match(pattern, email))

# models/user.py
from atoms.validators import is_valid_email

class User(Model):
    email = CharField(max_length=255, unique=True)

    def clean(self):
        if not is_valid_email(self.email):
            raise ValidationError("Invalid email format")
```

### Example 2: Model Using Molecule Helper

```python
# atoms/crypto.py
def hash_password(password: str) -> str:
    """Atom: Single-purpose hashing"""
    import bcrypt
    return bcrypt.hashpw(password.encode(), bcrypt.gensalt()).decode()

def verify_password(password: str, hashed: str) -> bool:
    """Atom: Single-purpose verification"""
    import bcrypt
    return bcrypt.checkpw(password.encode(), hashed.encode())

# molecules/password_handler.py
from atoms.crypto import hash_password, verify_password
from atoms.validators import validate_password_strength

class PasswordHandler:
    """Molecule: Combines hashing + validation"""

    @staticmethod
    def prepare_password(password: str) -> str:
        if not validate_password_strength(password):
            raise ValueError("Password too weak")
        return hash_password(password)

    @staticmethod
    def check_password(password: str, hashed: str) -> bool:
        return verify_password(password, hashed)

# models/user.py
from molecules.password_handler import PasswordHandler

class User(Model):
    email = CharField(max_length=255, unique=True)
    password_hash = CharField(max_length=255)

    def set_password(self, password: str):
        self.password_hash = PasswordHandler.prepare_password(password)

    def check_password(self, password: str) -> bool:
        return PasswordHandler.check_password(password, self.password_hash)
```

### Example 3: Model Associations

```python
# models/user.py
class User(Model):
    email = CharField(max_length=255, unique=True)
    created_at = DateTimeField(auto_now_add=True)

# models/post.py
class Post(Model):
    title = CharField(max_length=255)
    content = TextField()
    author = ForeignKey(User, on_delete=CASCADE, related_name='posts')
    created_at = DateTimeField(auto_now_add=True)

    class Meta:
        indexes = [
            Index(fields=['author', '-created_at']),  # Compound index
        ]

# Usage showing proper relationship navigation
user = User.objects.get(id=1)
posts = user.posts.all()  # Reverse relationship via related_name
```

### Example 4: Migration Using Atom Constants

```python
# atoms/constants.py
class UserRole:
    """Atom: Centralized enum values"""
    ADMIN = 'admin'
    MODERATOR = 'moderator'
    USER = 'user'

    CHOICES = [
        (ADMIN, 'Administrator'),
        (MODERATOR, 'Moderator'),
        (USER, 'Regular User'),
    ]

# migrations/0001_add_user_roles.py
from atoms.constants import UserRole

class Migration:
    def up(self):
        # Migration depends on atom constant
        add_column('users', 'role',
                   CharField(max_length=20,
                            choices=UserRole.CHOICES,
                            default=UserRole.USER))

    def down(self):
        remove_column('users', 'role')

# models/user.py
from atoms.constants import UserRole

class User(Model):
    email = CharField(max_length=255, unique=True)
    role = CharField(max_length=20,
                     choices=UserRole.CHOICES,
                     default=UserRole.USER)

    def is_admin(self) -> bool:
        return self.role == UserRole.ADMIN
```

## Verification Checklist

After creating or modifying database models:

- [ ] Model is organism-level (composes atoms/molecules, doesn't depend on other organisms)
- [ ] Validations use atom validators or molecule composite validators
- [ ] Constants come from atoms/constants.py, not hardcoded strings
- [ ] Associations use foreign keys, not direct code dependencies
- [ ] Migration includes both up AND down paths
- [ ] Indexes added for foreign keys and frequent queries
- [ ] Timestamps (created_at, updated_at) present on all tables
- [ ] Tests cover model methods, validations, and associations
- [ ] BV/BD bead status updated if working in beads mode
- [ ] No circular dependencies introduced (check with `bv --robot-insights`)
