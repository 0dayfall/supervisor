
# Go Concurrency Framework: Supervisor, Many-to-One, and GenStage Packages

![Go Version](https://img.shields.io/badge/Go-1.19-blue) ![License](https://img.shields.io/badge/license-MIT-green)

## Description

This project is a comprehensive Go framework that simplifies concurrent task management and data flow coordination. It comprises three main packages:
- **Supervisor**: Manages tasks with robust restart strategies, timeout handling, and recovery from panics.
- **Many-to-One**: Facilitates the coordination of data flow from multiple producers to a single consumer with data stitching and aggregation capabilities.
- **GenStage**: Provides a streamlined mechanism for data production, dispatching, and consumption with controlled rates and state management.

## Features

### Supervisor Package
- **Automated Task Supervision**: Restarts tasks with linear or exponential backoff strategies.
- **Error Recovery**: Handles errors and panics with built-in recovery mechanisms.
- **Timeout Capabilities**: Supports task execution with timeouts to prevent indefinite blocking.

### Many-to-One Package
- **Data Production and Aggregation**: Produces data using customizable functions and coordinates dispatch to a consumer.
- **Flexible Consumer Logic**: Processes data parts when all required pieces are present, with callback support for aggregation.
- **State Management**: Maintains and retrieves current states for producers and consumers.

### GenStage Package
- **Rate-Controlled Production and Consumption**: Producers and consumers operate with configurable production and consumption rates.
- **Dynamic Dispatching**: Dispatches data from a producer to a consumer based on demand.
- **Concurrent Data Handling**: Ensures safe state updates and data transfers between components.

## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
- [Supervisor Package Example](#supervisor-package-example)
- [Many-to-One Package Example](#many-to-one-package-example)
- [GenStage Package Example](#genstage-package-example)
- [Contributing](#contributing)
- [License](#license)

## Installation

Clone the repository and use Go modules to integrate the desired packages into your project:

```bash
git clone https://github.com/username/repo.git
```

Import the packages as needed:

```go
import "github.com/username/repo/supervisor"
import "github.com/username/repo/manytoone"
import "github.com/username/repo/genstage"
```

## Usage

### Supervisor Package Example

```go
task := func(ctx context.Context, msgChan chan Message) error {
    // Simulate task logic
    return nil
}

supervisor := supervisor.NewSupervisor(task, 3, 10*time.Second, 2*time.Second)
ctx := context.Background()
supervisor.Start(ctx)

// Stop the supervisor gracefully
supervisor.Stop()
```

### Many-to-One Package Example

```go
producer1 := manytoone.NewProducer[int](1, 100)
producer2 := manytoone.NewProducer[int](2, 100)

dispatcher := manytoone.NewDispatcher([]*manytoone.Producer[int]{producer1, producer2})
consumer := manytoone.NewConsumer(1, 5*time.Second, []string{"part1", "part2"}, func(data map[string]int) {
    log.Println("All parts received and processed:", data)
})

ctx := context.Background()
go dispatcher.Dispatch(ctx, consumer)
go consumer.Consume(ctx)

go producer1.Produce(ctx, func(i int) int { return i * 2 })
go producer2.Produce(ctx, func(i int) int { return i * 3 })
```

### GenStage Package Example

```go
producer := genstage.NewProducer[int](100, func(i int) int { return i * 2 }, 1*time.Second)
dispatcher := genstage.NewDispatcher(producer)
consumer := genstage.NewConsumer[int](1, func(data int) {
    log.Printf("Processing data: %d", data)
}, 2*time.Second)

ctx := context.Background()
go producer.Produce(ctx)
go dispatcher.Dispatch(ctx, consumer)
go consumer.Consume(ctx)
```

## Contributing

Contributions are welcome! Please see the `CONTRIBUTING.md` file for guidelines on how to contribute to this project.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
