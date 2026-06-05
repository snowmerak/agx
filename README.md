# agx

`agx` is a context-aware CLI wrapper for the `agy` development assistant tool. It binds active `agy` conversation sessions to the directory they were initialized in, providing seamless, directory-specific resume and prompt execution.

---

## Features

- **Directory-Bound Conversations**: Binds a single active conversation to your current working directory.
- **Automatic Initialization**: Runs programmatic initialization (`--auto`) in the background automatically if `agx` or `agx "prompt"` is executed in an uninitialized directory.
- **Explicit Initialization (`agx init`)**: Start a conversation once, interactively execute a system prompt, and automatically capture the generated conversation ID.
- **Programmatic Initialization (`agx init --auto`)**: Initialize a conversation mapping for the current directory headlessly in the background.
- **Seamless Resume (`agx`)**: Re-enter the interactive terminal session mapped to the current directory.
- **Fast Non-Interactive Execution (`agx "prompt"`)**: Run a single prompt directly against the mapped session and output the result.
- **Mapping Admin**: Easily inspect maps with `agx list` and clean them up with `agx remove <query>`.

---

## Installation

Ensure Go (1.20+) is installed and compile the executable:

```bash
go build -o agx main.go config.go
```

Move the compiled `agx` binary to a directory in your system's `PATH`.

---

## Configuration

Configuration is centrally stored in:
- **Windows**: `C:\Users\<Username>\.config\agx\config.json`
- **macOS/Linux**: `~/.config/agx/config.json`

The system prompt to initialize conversations can be configured in this JSON file under the `"system_prompt"` key.

---

## Usage

### 1. Initialize a Directory

#### Interactive Initialization
Before using `agx` in a new workspace, you can manually initialize it:
```bash
agx init
```
This launches `agy` with the pre-configured system prompt. Interact, get the response, and exit using `Control+D`. `agx` will automatically detect the conversation ID and map it to this directory.

#### Programmatic Initialization
To initialize headlessly without opening the interactive session:
```bash
agx init --auto
```

#### Automatic Initialization (Recommended)
You do not need to run `init` manually. If you execute a prompt or start the interactive session in an unmapped directory, `agx` will automatically initialize it in the background using your configured system prompt:
```bash
# In a new directory, this automatically initializes the session first:
agx "explain how this code works"

# Or start the interactive session directly:
agx
```

### 2. Resume Interactive Session
To return to the active interactive session for the current directory:
```bash
agx
```

### 3. Run a Single Prompt Non-Interactively
To send a prompt without opening the full TUI:
```bash
agx "explain how the current code works"
```

### 4. Manage Mappings

#### List all active mappings:
```bash
agx list
```

#### Remove a mapping:
```bash
agx remove <directory_path_or_conversation_id>
```
