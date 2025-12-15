## Coding style best practices

- **Consistent Naming Conventions**: Establish and follow naming conventions for variables, functions, classes, and files across the codebase
- **Automated Formatting**: Maintain consistent code style (indenting, line breaks, etc.)
- **Meaningful Names**: Choose descriptive names that reveal intent; avoid abbreviations and single-letter variables except in narrow contexts. Name things by what they ARE, not what they're used for—this enables reuse
- **Small, Focused Functions**: Keep functions small and focused on a single task for better readability and testability. Functions should be atoms (pure, no peer dependencies) or molecules (simple composition of 2-3 atoms). See `atomic-design.md`
- **Consistent Indentation**: Use consistent indentation (spaces or tabs) and configure your editor/linter to enforce it
- **Remove Dead Code**: Delete unused code, commented-out blocks, and imports rather than leaving them as clutter
- **Backward compatibility only when required:** Unless specifically instructed otherwise, assume you do not need to write additional code logic to handle backward compatibility.
- **DRY Principle**: Avoid duplication by extracting common logic into reusable functions or modules. However, require ≥3 real examples before abstracting—two duplications is acceptable, three warrants consideration. See `atomic-design.md`
