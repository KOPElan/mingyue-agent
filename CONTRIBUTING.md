# Contributing to Mingyue Agent

Thank you for your interest in contributing to Mingyue Agent! This document provides guidelines for contributing to the project.

## Code of Conduct

Please be respectful and constructive in all interactions. We aim to maintain a welcoming and inclusive community.

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- Clear, descriptive title
- Detailed steps to reproduce the issue
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages
- Screenshots if applicable

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, include:

- Clear, descriptive title
- Detailed description of the proposed functionality
- Use cases and benefits
- Possible implementation approach (optional)

### Pull Requests

1. **Fork the repository** and create your branch from `main` or `develop`
2. **Follow the coding standards** described below
3. **Write clear commit messages** following conventional commits format
4. **Add tests** for new functionality
5. **Update documentation** as needed
6. **Ensure all tests pass** before submitting
7. **Submit the pull request** with a clear description

## Development Setup

### Prerequisites

- Go 1.22 or higher
- Make
- Git
- Linux environment (for full functionality)

### Building

```bash
git clone https://github.com/KOPElan/mingyue-agent.git
cd mingyue-agent
make build
```

### Running Tests

```bash
make test
```

### Code Formatting

```bash
make fmt
make lint
```

## Coding Standards

### Go Guidelines

Follow the guidelines in `.github/instructions/go.instructions.md`:

1. **Naming Conventions**
   - Use mixedCaps for variables and functions
   - Use MixedCaps for exported names
   - Keep names short but descriptive
   - Avoid stuttering (e.g., http.Server not http.HTTPServer)

2. **Code Style**
   - Always use `gofmt` to format code
   - Keep line length reasonable
   - Add blank lines to separate logical groups
   - Write self-documenting code with clear names

3. **Error Handling**
   - Check errors immediately after function calls
   - Don't ignore errors using `_` without good reason
   - Wrap errors with context using `fmt.Errorf` with `%w`
   - Keep error messages lowercase and without punctuation

4. **Security**
   - All file operations must use PathValidator
   - All privileged operations must be audited
   - Input validation at every boundary
   - Follow principle of least privilege

### Package Structure

```
internal/
  api/          - HTTP/gRPC API handlers
  audit/        - Audit logging
  config/       - Configuration management
  daemon/       - Daemon lifecycle
  server/       - Server infrastructure
  filemanager/  - File operations
  diskmanager/  - Disk management
  monitor/      - System monitoring
```

### Testing

- Write unit tests for all new code
- Use table-driven tests for multiple test cases
- Name tests descriptively: `Test_functionName_scenario`
- Use subtests with `t.Run` for organization
- Test both success and error cases

### Documentation

- Document all exported types, functions, and packages
- Start documentation with the symbol name
- Use examples in documentation when helpful
- Keep documentation close to code
- Update documentation when code changes

### Commit Messages

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```
feat(diskmanager): add SMART monitoring support

Implements SMART health information reading via smartctl.
Includes temperature, power-on hours, and overall health status.

Closes #123

fix(api): prevent null pointer in disk handler

Added nil check for audit logger before logging disk operations.

docs(deployment): add Docker deployment guide

Created comprehensive deployment documentation including
systemd, Docker, and manual installation methods.
```

## Project-Specific Guidelines

### Adding New Features

1. Check IMPLEMENTATION.md for planned features
2. Create an issue to discuss the feature
3. Follow the existing architecture patterns
4. Integrate with audit logging
5. Add security validation where applicable
6. Update API documentation
7. Add tests

### Security Considerations

- All user input must be validated
- Use whitelist approach for paths and operations
- Log all privileged operations
- Follow least privilege principle
- Never execute arbitrary commands
- Use prepared statements for any database operations

### API Design

- Follow RESTful conventions
- Use structured JSON responses
- Return appropriate HTTP status codes
- Document all endpoints in docs/API.md
- Include examples in documentation
- Implement comprehensive error handling

## Review Process

1. All submissions require review
2. Address review feedback promptly
3. Keep pull requests focused and atomic
4. CI checks must pass before merge
5. Documentation must be updated

## Getting Help

- GitHub Discussions: https://github.com/KOPElan/mingyue-agent/discussions
- GitHub Issues: https://github.com/KOPElan/mingyue-agent/issues
- Documentation: docs/

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
