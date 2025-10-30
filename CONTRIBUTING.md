# Contributing to GUUID

Thank you for your interest in contributing to GUUID! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git
- Make (optional, but recommended)

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/guuid.git
   cd guuid
   ```

3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/lab2439/guuid.git
   ```

4. Install dependencies:
   ```bash
   go mod download
   ```

## Development Workflow

### Running Tests

```bash
# Run all tests
make test

# Run tests with race detector
go test -race ./...

# Run short tests only
make test-short

# Generate coverage report
make coverage
```

### Running Benchmarks

```bash
make bench
```

### Code Quality

```bash
# Format code
make fmt

# Run linters
make lint

# Run all checks (fmt, vet, lint, test)
make check
```

### Building Examples

```bash
make build
```

## Coding Standards

### Code Style

- Follow standard Go code style
- Use `gofmt` to format your code
- Run `go vet` to check for common mistakes
- Pass all `golangci-lint` checks

### Documentation

- Add comments for all exported functions, types, and constants
- Use complete sentences in comments
- Start comments with the name of the element being documented
- Include examples for complex functionality

### Testing

- Write tests for all new functionality
- Maintain or improve code coverage
- Include both positive and negative test cases
- Use table-driven tests where appropriate
- Add benchmarks for performance-critical code

### Commit Messages

Follow the conventional commits specification:

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
- `perf`: Performance improvements

Example:
```
feat(v7): add batch generation support

Add a new method to generate multiple UUIDs in a single call,
improving performance for bulk operations.

Closes #123
```

## Pull Request Process

1. Create a new branch for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit them with clear commit messages

3. Update documentation as needed

4. Add or update tests to cover your changes

5. Ensure all tests pass:
   ```bash
   make check
   ```

6. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

7. Create a pull request to the upstream repository

8. Wait for review and address any feedback

### Pull Request Guidelines

- Keep pull requests focused on a single feature or fix
- Include tests for new functionality
- Update documentation as needed
- Ensure CI checks pass
- Link related issues in the PR description

## Reporting Issues

### Bug Reports

When reporting bugs, please include:

- Go version
- Operating system
- Minimal code to reproduce the issue
- Expected behavior
- Actual behavior
- Stack trace (if applicable)

### Feature Requests

When requesting features, please include:

- Use case description
- Proposed API (if applicable)
- Any alternatives you've considered

## Code Review

All submissions require review. We use GitHub pull requests for this purpose.

Reviewers will look for:

- Code quality and style
- Test coverage
- Documentation
- Performance implications
- Backward compatibility

## License

By contributing to GUUID, you agree that your contributions will be licensed under the MIT License.

## Questions?

Feel free to open an issue for any questions or concerns.

