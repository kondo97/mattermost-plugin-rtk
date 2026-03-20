# Technology Stack

## Programming Languages
- Go 1.25 - Backend plugin implementation
- TypeScript - Frontend (React components, type-safe development)
- JavaScript - webpack/Babel configuration

## Frameworks
- Mattermost Plugin SDK (`mattermost/server/public` v0.1.21) - Plugin interface
- gorilla/mux v1.8.1 - Go HTTP router
- React - Frontend UI (via `@mattermost/types` 11.1.0)
- Redux - Frontend state management (RTK: Redux Toolkit)
- Emotion (`@emotion/react` 11.9.0) - CSS-in-JS

## Infrastructure
- Mattermost Server - Plugin host environment
- Mattermost KV Store - Data persistence
- Mattermost Cluster (`pluginapi/cluster`) - Distributed job scheduling

## Build Tools
- Make - Integrated build system (build/deploy/release)
- Go modules (`go.mod`) - Go dependency management
- npm v8 / Node.js v16+ - Frontend dependency management
- webpack 5 - Frontend bundler
- Babel 7 - JavaScript/TypeScript transpiler

## Testing Tools
- Go test (`testing` package) - Backend test framework
- testify v1.11.1 - Go test assertions
- go-mock (go.uber.org/mock v0.6.0) - Go mock generation
- Jest - Frontend test framework
- @testing-library/jest-dom 5.16.1 - Jest DOM assertions
- Enzyme - React component testing

## Linting / Code Quality
- golangci-lint - Go code quality checks
- ESLint - TypeScript/JavaScript linter (`@mattermost/eslint-plugin`)
- TypeScript compiler (`tsc`) - Type checking

## CI/CD
- GitHub Actions (`.github/workflows/ci.yml`, `e2e.yml` - referenced in README)
