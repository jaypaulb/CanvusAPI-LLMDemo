# Contributing to CanvusAPI-LLMDemo

Thank you for your interest in contributing to CanvusAPI-LLMDemo! This document provides guidelines and instructions for contributing.

## Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/CanvusAPI-LLMDemo.git
   ```
3. Create a new branch for your feature:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. Copy `example.env` to `.env` and configure your environment variables
5. Install dependencies:
   ```bash
   go mod download
   ```

## Code Style Guidelines

- Follow standard Go formatting guidelines (use `gofmt`)
- Add comments for exported functions and packages
- Keep functions focused and concise
- Use meaningful variable and function names
- Include error handling for all operations that could fail

## Testing

- Write tests for new features
- Ensure existing tests pass
- Include both unit tests and integration tests where appropriate
- Test error conditions and edge cases

## Commit Guidelines

- Use clear, descriptive commit messages
- Start with a verb in the present tense (e.g., "Add feature" not "Added feature")
- Reference issue numbers in commits where applicable
- Keep commits focused and atomic

## Pull Request Process

1. Update documentation for any new features
2. Ensure all tests pass
3. Update the README.md if needed
4. Create a pull request with a clear description of changes
5. Link any relevant issues
6. Wait for review and address any feedback

## Reporting Issues

When reporting issues, please include:

- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages

## Security Issues

For security issues, please email directly instead of creating a public issue.

## License

By contributing, you agree that your contributions will be licensed under the project's proprietary license. 