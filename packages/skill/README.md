# @cloo-solutions/neotexai

Claude Code skill for [neotex](https://github.com/cloo-solutions/neotexai) - agent-first knowledge management.

## Installation

```bash
npm install @cloo-solutions/neotexai
```

The skill is automatically installed to `~/.claude/skills/neotex/` on npm install.

## What It Does

This skill teaches Claude Code how to:

- **Search** organizational knowledge when you ask about conventions, guidelines, or past decisions
- **Store** learnings and decisions after completing significant work
- **Manage assets** like reference images, mockups, and design files

## Prerequisites

You need the neotex CLI installed and configured:

```bash
# Install neotex CLI (Go)
go install github.com/cloo-solutions/neotexai/cmd/neotex@latest

# Initialize a project
neotex init

# Set your API key
export NEOTEX_API_KEY=your_key
```

## Manual Installation

If automatic installation didn't work:

```bash
neotex skill init
```

## Usage

Once installed, Claude Code automatically uses neotex when relevant. Examples:

- "How do we handle authentication here?" → searches knowledge base
- "Save this as a guideline" → stores to neotex
- *Upload a mockup* → offers to save as asset

## License

MIT
