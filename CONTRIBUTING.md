# Contributing to NeotexAI

Thank you for your interest in contributing to NeotexAI! This document provides guidelines for contributing to the project.

## Code of Conduct

This project adheres to the Contributor Covenant [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Bugs

- Use the GitHub issue tracker
- Check if the issue has already been reported
- Use the bug report template
- Include steps to reproduce, expected behavior, and actual behavior
- Include your environment details (OS, Go version, etc.)

### Suggesting Enhancements

- Use the GitHub issue tracker
- Use the feature request template
- Clearly describe the feature and its use case
- Explain why this enhancement would be useful

### Pull Requests

1. Fork the repository
2. Create a new branch (`git checkout -b feature/your-feature-name`)
3. Make your changes
4. Write or update tests as needed
5. Ensure all tests pass (`go test ./...`)
6. Run `go fmt` and `go vet`
7. Commit your changes with clear, descriptive messages
8. Push to your fork
9. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/cloo-solutions/neotexai.git
cd neotexai

# Install dependencies
go mod download

# Run tests
go test ./...

# Build binaries
make build
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write clear, descriptive variable and function names
- Add comments for exported functions and types
- Keep functions focused and concise

## Testing

- Write tests for new features
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Include both positive and negative test cases

## Commit Messages

- Use clear, descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Reference issue numbers when applicable
- Keep the first line under 72 characters

## License

By contributing to NeotexAI, you agree that your contributions will be licensed under the project's OSAASy License.

## Questions?

Feel free to open an issue for any questions about contributing.
