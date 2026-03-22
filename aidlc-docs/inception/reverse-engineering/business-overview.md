# Business Overview

## Business Context Diagram

```
+-------------------------------------------------------+
|                  Mattermost Server                    |
|                                                       |
|  +---------------------------------------------------+|
|  |          Plugin: mattermost-plugin-rtk            ||
|  |                                                   ||
|  |  +---------------+    +-----------------------+  ||
|  |  |  Server (Go)  |    |   Webapp (React/TS)   |  ||
|  |  | - HTTP API    |    | - UI Components       |  ||
|  |  | - Slash Cmds  |    | - Redux Store         |  ||
|  |  | - KV Store    |    | - Plugin Registry     |  ||
|  |  | - Background  |    |                       |  ||
|  |  |   Job         |    |                       |  ||
|  |  +---------------+    +-----------------------+  ||
|  +---------------------------------------------------+|
+-------------------------------------------------------+
          |                          |
          v                          v
  [Mattermost Users]          [Mattermost Channels]
```

## Business Description

- **Business Description**: This repository is a Mattermost plugin starter template. It provides a foundation (boilerplate) for building custom plugins that extend the Mattermost platform. Currently, it serves as a starting point for developing a Mattermost plugin with a frontend leveraging RTK (Redux Toolkit).

- **Business Transactions**:
  1. **Plugin Activation**: The Mattermost server loads and activates the plugin, initializing the HTTP router, command handler, KV store, and background job.
  2. **Slash Command Execution (/hello)**: A user types `/hello @username` and the plugin returns a greeting message.
  3. **HTTP API Call (/api/v1/hello)**: An authenticated client calls the REST API and receives a "Hello, world!" response.
  4. **KV Store Access**: The plugin stores and retrieves user-specific data via the Mattermost KV store.
  5. **Background Job Execution**: A cluster-scheduled job runs every hour.

- **Business Dictionary**:
  - **Plugin**: An extension module dynamically loaded into the Mattermost server
  - **KV Store**: Plugin-specific key-value storage provided by Mattermost
  - **Slash Command**: A command prefixed with `/` entered in a Mattermost channel
  - **Hook**: A plugin callback function triggered by Mattermost server events
  - **pluginapi**: The official Go client library for the Mattermost Plugin API
  - **RTK**: Redux Toolkit (frontend state management library)

## Component Level Business Descriptions

### Server (Go Backend)
- **Purpose**: The plugin server-side core. Handles all server-side business logic.
- **Responsibilities**: Plugin lifecycle management, HTTP API serving, slash command processing, data persistence, background job execution.

### Webapp (React/TypeScript Frontend)
- **Purpose**: Renders plugin-specific UI components within the Mattermost interface.
- **Responsibilities**: Plugin UI registration, Redux Store integration, user interaction handling.

### command package
- **Purpose**: Isolates slash command registration and execution logic.
- **Responsibilities**: Registering and executing the `/hello` command.

### kvstore package
- **Purpose**: Abstracts access to the Mattermost KV Store.
- **Responsibilities**: Wrapping KV Store methods, improving testability.
