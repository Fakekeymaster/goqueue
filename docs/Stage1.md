# Stage 1 - Project skeleton + go mod init

**Before Stage 1 — two quick concepts you need**
*Goroutines vs threads:*
A thread in C++ costs ~1MB of stack. A goroutine costs ~8KB. You can run 100,000 goroutines easily. Our worker pool is just N goroutines — extremely lightweight.
*Channels:*
Go's way of passing data between goroutines safely. Think of it like a pipe — one goroutine writes in, another reads out. We'll use channels for the shutdown signal.

## Here's what stage 1 involves:
- mkdir goqueue && cd goqueue
- go mod init github.com/yourusername/goqueue
- Create the folder Structure 
- Verify it builds with an empty main.go 

### Folder Structure
goqueue/
├── main.go         → entry point, just calls cmd
├── cmd/            → CLI commands (server, submit, status, stats)
├── queue/          → Job struct and types
├── worker/         → goroutine pool and worker logic
├── api/            → REST API server
├── store/          → Redis operations
└── config/         → app configuration

This is called spearation of concerns