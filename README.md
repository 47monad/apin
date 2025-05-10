# apin

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/47monad/apin)](https://goreportcard.com/report/github.com/47monad/apin)
[![Go Reference](https://pkg.go.dev/badge/github.com/47monad/apin.svg)](https://pkg.go.dev/github.com/47monad/apin)

> **apin** (/æpin/) - short for "**ap**p **in**itializer" - A toolkit for bootstrapping Go applications with elegance and simplicity.

## Overview

Apin is a lightweight, modular framework designed to streamline the initialization and bootstrapping process of Go applications. It provides a structured approach to setting up application components, managing dependencies, and handling service lifecycles.

## Features

- **Modular Architecture**: Define your application as a collection of independent modules
- **Dependency Management**: Declare and inject dependencies between application components
- **Lifecycle Control**: Graceful startup and shutdown of services
- **Configuration Integration**: Seamless handling of configuration from different sources
- **Context Propagation**: Proper context handling throughout your application
- **Middleware Support**: Easily add cross-cutting concerns to your application
- **Testing Friendly**: Simple approach to mocking and testing components

## Installation

```bash
go get github.com/47monad/apin
```

## Quick Start

```go
package main

import (
 "context"
 "log"
 "time"

 "github.com/47monad/apin"
)

func main() {
 // Create a new app initializer
 app := apin.New()

 // Register modules
 app.Register(
  // Configuration module
  NewConfigModule(),
  // Database module
  NewDatabaseModule(),
  // Service module
  NewServiceModule(),
 )

 // Run the application with a timeout context
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()

 if err := app.Run(ctx); err != nil {
  log.Fatalf("application failed: %v", err)
 }
}

// Example module implementation
type ConfigModule struct {
 // Module fields
}

func NewConfigModule() *ConfigModule {
 return &ConfigModule{}
}

func (m *ConfigModule) Init(ctx context.Context) error {
 // Initialize configuration
 return nil
}

func (m *ConfigModule) Name() string {
 return "config"
}
```

## Core Concepts

### Modules

Modules are the building blocks of your application. Each module represents a discrete piece of functionality:

```go
// Module is the interface that all application modules must implement
type Module interface {
 // Init initializes the module
 Init(ctx context.Context) error
 
 // Name returns the module name
 Name() string
}
```

### Application Lifecycle

Apin manages your application's lifecycle through these phases:

1. **Registration**: Register all modules
2. **Initialization**: Modules are initialized in dependency order
3. **Running**: The application runs until interrupted or context cancelled
4. **Shutdown**: Modules are shut down in reverse order

### Dependency Management

Define dependencies between modules to control initialization order:

```go
// DatabaseModule depends on ConfigModule
type DatabaseModule struct {
 Config *ConfigModule
}

func NewDatabaseModule() *DatabaseModule {
 return &DatabaseModule{}
}

func (m *DatabaseModule) Init(ctx context.Context) error {
 // Use m.Config which was injected automatically
 return nil
}

func (m *DatabaseModule) Name() string {
 return "database"
}

func (m *DatabaseModule) Requires() []string {
 return []string{"config"}
}
```

## Advanced Usage

### Graceful Shutdown

Implement the `Shutdown` interface to manage resource cleanup:

```go
type ServerModule struct {
 server *http.Server
}

func (m *ServerModule) Init(ctx context.Context) error {
 // Initialize HTTP server
 return nil
}

func (m *ServerModule) Shutdown(ctx context.Context) error {
 // Gracefully shutdown HTTP server
 return m.server.Shutdown(ctx)
}
```

### Configuration Integration

Integrate with your configuration management solution:

```go
app := apin.New(
 apin.WithConfig(&myConfig),
)
```

## Example Applications

See the [examples](https://github.com/47monad/apin/tree/main/examples) directory for complete working applications using apin:

- Basic HTTP API server
- gRPC service
- Worker application
- CLI tool

## Philosophy

Apin follows these guiding principles:

1. **Simplicity over complexity**: Straightforward APIs with minimal overhead
2. **Explicit over implicit**: Clear declaration of dependencies and initialization order
3. **Composition over inheritance**: Build applications from composable modules
4. **Standard library first**: Leverage the Go standard library when possible
5. **Testing first**: Design for testability from the ground up

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with ❤️ by [47monad](https://github.com/47monad)

