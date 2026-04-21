# Build Instructions

## Prerequisites

- **Go**: 1.21+ (`go version`)
- **Node.js**: 18+ (`node -v`)
- **npm**: 9+ (`npm -v`)
- **Make**: GNU Make (`make --version`)
- **Environment Variables**: None required for local build
- **System Requirements**: macOS or Linux; 4GB+ RAM

## Build Steps

### 1. Install Dependencies

```bash
# Install frontend dependencies
cd webapp && npm install && cd ..
```

### 2. Install Go Tools

```bash
make install-go-tools
# Installs: golangci-lint v2.9.0, gotestsum v1.13.0
```

### 3. Build All Units (Full Distribution Bundle)

```bash
make dist
```

This runs the following sequence automatically:

| Step | Command | Description |
|------|---------|-------------|
| 1 | `make apply` | Propagate plugin.json metadata to server/ and webapp/ |
| 2 | `make webapp` | Build React/TypeScript frontend with Vite |
| 3 | `make copy-call-js` | Copy `webapp/dist/call.js` to `server/assets/call.js` for go:embed |
| 4 | `make server` | Build Go server for all target platforms |
| 5 | `make bundle` | Package everything into a .tar.gz bundle |

### 4. Verify Build Success

- **Expected Output**: `plugin built at: dist/com.kondo97.mattermost-plugin-rtk-{version}.tar.gz`
- **Build Artifacts**:
  - `webapp/dist/main.js` — Frontend bundle
  - `webapp/dist/call.js` — RTK call page bundle
  - `server/dist/plugin-linux-amd64`
  - `server/dist/plugin-linux-arm64`
  - `dist/com.kondo97.mattermost-plugin-rtk-{version}.tar.gz`

### 5. Developer Build (Current Platform Only)

```bash
# Build only for your current OS/arch (faster)
MM_SERVICESETTINGS_ENABLEDEVELOPER=1 make dist
```

### 6. Style Check (Optional, Pre-commit)

```bash
make check-style
# Runs: eslint, tsc, go vet, golangci-lint
```

### 7. Clean Build Artifacts

```bash
make clean
```

## Troubleshooting

### Build Fails: `webapp/dist/call.js not found`

- **Cause**: `make copy-call-js` ran before `make webapp` completed.
- **Solution**: Run `make dist` (not individual steps) to ensure correct order.

### Build Fails: Go Compilation Error

- **Cause**: Missing `server/assets/call.js` (go:embed target).
- **Solution**: Run `make webapp && make copy-call-js` before `make server`.

### Build Fails: `golangci-lint: command not found`

- **Cause**: Go tools not installed.
- **Solution**: Run `make install-go-tools` first.

### Frontend Build Fails: TypeScript Errors

- **Cause**: Type errors in webapp/src/.
- **Solution**: Run `cd webapp && npm run check-types` to identify errors.

### npm install Fails

- **Cause**: Node.js version incompatibility.
- **Solution**: Use Node.js 18+. Consider using `nvm use 18`.
