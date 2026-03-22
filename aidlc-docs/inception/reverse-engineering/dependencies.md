# Dependencies

## Internal Dependencies

```
server/ (main)
  +-- server/command/        (command.Command interface)
  +-- server/store/kvstore/  (kvstore.KVStore interface)

server/command/
  (no internal deps)

server/store/kvstore/
  (no internal deps)

webapp/src/
  (no internal deps)
```

### server/ depends on server/command/
- **Type**: Compile
- **Reason**: Plugin delegates slash command handling to the command package

### server/ depends on server/store/kvstore/
- **Type**: Compile
- **Reason**: Plugin abstracts KV store access via the kvstore package

## External Dependencies (Go)

### github.com/mattermost/mattermost/server/public
- **Version**: v0.1.21
- **Purpose**: Mattermost Plugin SDK - plugin hooks, pluginapi client, model types, cluster scheduling
- **License**: Apache-2.0

### github.com/gorilla/mux
- **Version**: v1.8.1
- **Purpose**: HTTP routing
- **License**: BSD-3-Clause

### github.com/pkg/errors
- **Version**: v0.9.1
- **Purpose**: Error wrapping and context attachment
- **License**: BSD-2-Clause

### github.com/stretchr/testify
- **Version**: v1.11.1
- **Purpose**: Test assertions (assert/require)
- **License**: MIT

### go.uber.org/mock
- **Version**: v0.6.0
- **Purpose**: Auto-generation of interface mocks
- **License**: Apache-2.0

## External Dependencies (npm - devDependencies)

### @mattermost/types
- **Version**: 11.1.0
- **Purpose**: Mattermost TypeScript type definitions
- **License**: MIT

### @mattermost/client
- **Version**: 11.1.0
- **Purpose**: Mattermost client API
- **License**: MIT

### react / react-dom
- **Version**: (defined as externals in webpack.config.js, provided by Mattermost host)
- **Purpose**: UI framework
- **License**: MIT

### redux
- **Version**: (provided by Mattermost host)
- **Purpose**: State management
- **License**: MIT

### @emotion/react
- **Version**: 11.9.0
- **Purpose**: CSS-in-JS styling
- **License**: MIT

### webpack / babel
- **Version**: webpack 5, Babel 7
- **Purpose**: Build toolchain
- **License**: MIT
