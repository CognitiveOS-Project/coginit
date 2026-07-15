# CognitiveOS Init (coginit)

Unified PID 1 process supervisor for CognitiveOS across Docker and bare-metal environments.

## What this does

`coginit` is a compiled Go binary that replaces the fragile shell-based init chain:

- **Docker**: replaces `tini` + `docker-init.sh`
- **Bare-metal**: coexists with OpenRC, handles CognitiveOS services only

## Architecture

```
coginit (PID 1)
  │
  │  1. Install boot-stage dependencies (cpm)
  │  2. Start cograw → wait raw.sock
  │  3. Start coginfer → wait HTTP health
  │  4. Start cognitiveosd → wait daemon.sock
  │  5. Install runtime-stage dependencies (cpm)
  │  6. Exec CLI (Docker) / Supervise CLI (bare-metal)
```

## Build

```bash
# Development build
make build

# Static binary for Docker/ISO
make build-static
```

## Usage

```bash
# Auto-detect environment
./coginit

# Force Docker mode
./coginit --docker

# Force bare-metal mode
./coginit --bare-metal

# Show version
./coginit --version
```

## Integration

- **Docker**: `COPY --from=builder /path/to/coginit /usr/local/bin/coginit`
- **ISO**: `tty1::respawn:/usr/local/bin/coginit` in inittab
