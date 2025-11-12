# Notification System - Task 2 Implementation

A high-performance, scalable notification delivery system built with Go, capable of handling **40,000+ notifications per minute** with priority-based processing, intelligent batching, and robust retry mechanisms.

## 📋 Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Approach](#approach)
- [Technology Stack](#technology-stack)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the System](#running-the-system)
- [Database Setup](#database-setup)
- [System Design](#system-design)
- [Performance Metrics](#performance-metrics)
- [Testing](#testing)
- [Future Enhancements](#future-enhancements)

## ✨ Features

### Core Functionality
- ✅ **Priority-based Processing**: 4-level priority system (URGENT, HIGH, MEDIUM, LOW)
- ✅ **Multi-channel Delivery**: Support for EMAIL, PUSH, IN_APP, and SMS notifications
- ✅ **Intelligent Batching**: Automatic batching of low-priority notifications
- ✅ **Retry Mechanism**: Exponential backoff retry strategy with configurable attempts
- ✅ **Dead Letter Queue (DLQ)**: Failed notifications tracking for manual intervention
- ✅ **User Preferences**: Per-user, per-channel notification settings
- ✅ **Quiet Hours**: Respect user-defined quiet hours
- ✅ **High Throughput**: Handles 40,000+ notifications per minute

### Technical Features
- ✅ **Concurrent Processing**: Worker pool pattern with priority-based allocation
- ✅ **Database Persistence**: PostgreSQL with pgx/v5 for data storage
- ✅ **UUID v5**: Deterministic UUID generation for notifications
- ✅ **Graceful Shutdown**: Proper cleanup and resource management
- ✅ **Metrics & Monitoring**: Real-time processing statistics
- ✅ **Structured Logging**: JSON-formatted logs for observability

## 🏗 Architecture

### High-Level Architecture

```
[Ingestion]
  └── NotificationService (validate, preferences, quiet hours)
        ├── Repository (persist, fetch)
        └── Queue (enqueue by priority)

[Processing]
  └── Processor (worker pool)
        └── Strategy
             ├── Priority (allocate workers)
             ├── Batch (group low-priority)
             └── Retry (exponential backoff, DLQ)

[Delivery]
  └── Channel Drivers (e.g., Firebase Push)
        └── External Providers

[Observability]
  ├── Logger (structured logs)
  └── Metrics (processing stats)
```

## 🧭 Approach

### Goals
- Deliver 40k+ notifications/min while preserving reliability and user experience.
- Enforce priority, retries, and DLQ to prevent data loss.
- Respect per-user preferences and quiet hours before sending.
- Keep architecture modular so channels and strategies are pluggable.

### Design Decisions
- Worker pool with priority-aware allocation ensures urgent items are processed first.
- Buffer/queue decouples ingestion from delivery and absorbs load spikes.
- Strategies implemented in `internal/strategy` keep logic testable and configurable.
- PostgreSQL repository layer provides durability and supports analytics.
- Channel drivers under `internal/delivery` (Firebase push implemented) enable easy expansion.

### Processing Flow
- Ingestion: `internal/service/notification_service.go` validates input, checks preferences/quiet hours, assigns deterministic IDs, and persists via `internal/repository`.
- Enqueue: Notifications are pushed to `internal/queue` with priority tagging.
- Dispatch: `internal/processor` runs workers sized per priority using `internal/strategy/priority.go`.
- Batching: `internal/strategy/batch.go` groups low-priority items to reduce provider overhead.
- Delivery: `internal/delivery` sends via the appropriate channel (Firebase push driver available).
- Retry & DLQ: `internal/strategy/retry.go` applies exponential backoff; exceeding attempts moves to DLQ.
- Logging & Metrics: `pkg/logger` emits structured logs; hooks track throughput and failures.
- Shutdown: Context-aware workers drain safely and release resources.

### Core Algorithms
- Priority Allocation: Configurable ratios give URGENT/HIGH most capacity while fairly serving MEDIUM/LOW.
- Batching Policy: For low-priority queues, collect up to batch size or timeout, then send as a group.
- Exponential Backoff: Add jitter to avoid thundering herds; non-retryable errors bypass retries.

### Throughput Strategy (40k+/min)
- Size the worker pool based on CPU/network capacity and provider rate limits.
- Use efficient DB operations and connection pooling (pgx/v5).
- Batch low-priority sends to reduce external API calls.
- Decouple ingestion and processing to absorb bursts without drops.

### Failure Handling
- Retries for transient errors; classify non-retryable errors to skip immediately.
- Record failures and final outcomes in DLQ for inspection and reprocessing.
- Structured logs provide traceability per notification and channel.

### Extensibility
- Add new channels by implementing a driver in `internal/delivery` and wiring it in the service.
- Adjust priority ratios, batch sizes, and retry settings via `internal/config`.
- Extend preferences and quiet hours with richer policy rules.

### How This Solves the Assignment
- Meets priority-based processing with dedicated worker allocation.
- Achieves high throughput using a decoupled queue, worker pool, and batching.
- Ensures reliability with retries and DLQ.
- Respects user preferences and quiet hours ahead of delivery.
- Maintains clean separation of concerns for maintainability and future growth.

