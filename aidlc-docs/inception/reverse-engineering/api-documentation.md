# API Documentation

## REST APIs

### Hello World
- **Method**: GET
- **Path**: `/plugins/com.mattermost.plugin-starter-template/api/v1/hello`
- **Purpose**: Sample endpoint. Returns "Hello, world!"
- **Authentication**: Requires authentication via `Mattermost-User-ID` header
- **Request**: None (no query parameters)
- **Response**: `200 OK` - Body: `"Hello, world!"`
- **Error Response**: `401 Unauthorized` - when `Mattermost-User-ID` header is missing

## Slash Commands

### /hello
- **Trigger**: `/hello`
- **Purpose**: Sends a greeting message to a specified user
- **Usage**: `/hello [@username]`
- **AutoComplete**: Enabled
- **Response Type**: Ephemeral (visible only to the command executor)
- **Examples**:
  - `/hello @alice` → "Hello, @alice"
  - `/hello` (no argument) → "Please specify a username"

## Internal APIs (Go Interfaces)

### Command Interface
```
Interface: command.Command
Package:   server/command

Methods:
  Handle(args *model.CommandArgs) (*model.CommandResponse, error)
    - Processes a slash command and returns a response
    - args: Mattermost command arguments (Command string, UserId, ChannelId, etc.)
    - Returns: CommandResponse (Text, ResponseType) or error

  executeHelloCommand(args *model.CommandArgs) *model.CommandResponse
    - Executes the /hello command
    - args: Command arguments
    - Returns: CommandResponse with greeting text
```

### KVStore Interface
```
Interface: kvstore.KVStore
Package:   server/store/kvstore

Methods:
  GetTemplateData(userID string) (string, error)
    - Retrieves user-specific template data from the KV store
    - userID: Mattermost user ID
    - Key pattern: "template_key-{userID}"
    - Returns: stored string data or error
```

### Plugin Hooks (Mattermost SDK)
```
Implemented Hooks on Plugin struct:

  OnActivate() error
    - Called when the plugin is activated
    - Initializes: client, kvstore, commandClient, router, backgroundJob

  OnDeactivate() error
    - Called when the plugin is deactivated
    - Cleans up backgroundJob

  ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError)
    - Slash command execution hook
    - Delegates to commandClient.Handle()

  ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request)
    - HTTP request handling hook
    - Delegates to gorilla/mux router

  OnConfigurationChange() error
    - Called when configuration may have changed
    - Reloads and updates configuration
```

## Data Models

### configuration (struct)
- **Fields**: Currently empty (no configuration fields)
- **Relationships**: Held by the Plugin struct
- **Validation**: None (no fields)
- **Note**: New configuration values should be added alongside `settings_schema` in `plugin.json`

### model.CommandResponse (Mattermost)
- **Text**: Response text
- **ResponseType**: `"in_channel"` (visible to all) or `"ephemeral"` (visible to executor only)

### KV Store Key Patterns
- `template_key-{userID}` - User-specific template data
