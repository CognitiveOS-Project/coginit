# coginit

Unified PID 1 process supervisor for CognitiveOS across Docker and bare-metal environments.

Replaces the fragile shell-based init chain (`tini` + `docker-init.sh` for Docker, `busybox init` + OpenRC for bare-metal) with a single, auditable Go binary.

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

### Project Layout

```
cmd/coginit/main.go           — Entry point, flag parsing
internal/coginit/
  init.go                     — Run(), Version(), IsDocker()
  engine.go                   — Engine start/wait orchestration, log rotation
  socket.go                   — Socket and HTTP health polling
  init_docker.go              — Docker mode: exec CLI after engines start
  init_baremetal.go           — Bare-metal mode: mount FS, loopback, TUI supervision
  backdoor.go                 — VT + serial backdoor monitor (key combos)
  process.go                  — Child process tracking, process group management
  signal.go                   — SIGCHLD reaper, SIGINT/SIGTERM shutdown
  vt.go                       — Virtual terminal switching, evdev keyboard detection
  *_stub.go                   — No-op stubs for non-Linux platforms
```

## Boot Sequence

### Docker Mode

1. Start signal reaper (zombie reaping)
2. Create runtime directories (`/cognitiveos/run`, `/cognitiveos/logs`)
3. Install boot-stage dependencies via `cpm install-dependencies --stage boot`
4. Start engines in order: `cograw` → `coginfer` → `cognitiveosd`
5. Wait for each engine's readiness (socket or HTTP health)
6. Install runtime-stage dependencies via `cpm install-dependencies --stage runtime`
7. Exec `cognitiveos-cli` (replaces coginit process)

### Bare-Metal Mode

1. Mount virtual filesystems (`/proc`, `/sys`, `/tmp`, `/run`, `/dev/pts`)
2. Start signal reaper (zombie reaping + power off on shutdown)
3. Configure loopback network interface
4. Create runtime directories
5. Install boot-stage dependencies
6. Start engines in order (same as Docker)
7. Install runtime-stage dependencies
8. Start backdoor monitor (keyboard combos + serial console)
9. TUI supervision loop (restarts CLI on crash)

## Backdoor Shell

Documented in [ADR-003](https://github.com/CognitiveOS-Project/product-specs/blob/main/adr/ADR-003-backdoor-shell.md). Activated via hardware key combos on VT keyboard or serial console.

| Combo | Trigger |
|-------|---------|
| Ctrl+Alt+Backspace | VT keyboard |
| Alt+PrintScreen+K | VT keyboard |
| Ctrl+Alt+Del | VT keyboard |
| ScrollLock ×3 (within 2s) | VT keyboard |
| Super+Grave | VT keyboard |
| Serial activity | `/dev/ttyAMA0` |

Spawns `/bin/sh` on `/dev/tty2` (VT) or the serial TTY. Access is logged to `/cognitiveos/logs/backdoor.log`.

## Build

```bash
make build         # compile to build/bin/coginit
make build-static  # static binary for Docker/ISO (amd64 + arm64)
make test          # run tests
make lint          # go vet
make clean         # remove build artifacts
```

## Usage

```bash
# Auto-detect environment
coginit

# Force Docker mode
coginit --docker

# Force bare-metal mode
coginit --bare-metal

# Show version
coginit --version
```

## Integration

**Docker:**
```dockerfile
COPY --from=builder /path/to/coginit /usr/local/bin/coginit
ENTRYPOINT ["coginit"]
```

**ISO / bare-metal inittab:**
```
tty1::respawn:/usr/local/bin/coginit
```

## Configuration

coginit reads no configuration files. All paths are hardcoded to CognitiveOS conventions:

| Path | Purpose |
|------|---------|
| `/cognitiveos/run/` | Socket files (raw.sock, daemon.sock) |
| `/cognitiveos/logs/` | Service logs with rotation (10MB limit) |
| `/cognitiveos/models/` | GGUF model weights |
| `/usr/local/bin/` | Engine binaries (cograw, coginfer, cognitiveosd, cpm) |

## Environment Detection

- **Docker**: `/.dockerenv` or `/run/.containerenv` exists, or `container` env var is set
- **Bare-metal**: everything else

## Related

- [CognitiveOS](https://github.com/CognitiveOS-Project/cognitiveos) — main project repository
- [cognitive-os.org](https://cognitive-os.org) — project website
- [inference](https://github.com/CognitiveOS-Project/inference) — cograw + coginfer engines
- [cognitiveosd](https://github.com/CognitiveOS-Project/cognitiveosd) — system daemon
- [cli](https://github.com/CognitiveOS-Project/cli) — TUI frontend
- [cpm](https://github.com/CognitiveOS-Project/cpm) — CognitiveOS package manager
- [Product Specs](https://github.com/CognitiveOS-Project/product-specs) — boot flow specification
- [CognitiveOS Project](https://github.com/CognitiveOS-Project) — GitHub organization

## Contributing

1. Branch from `main`
2. Use topic branches: `feature/<name>`, `fix/<name>`
3. Open a PR to `main` with a clear title and description
4. Merge after review

See the [SDLC repo](https://github.com/CognitiveOS-Project/sdlc) for the full contribution guide, code review standards, and testing strategy.

## Author

**Jean Machuca** — [GitHub](https://github.com/jeanmachuca) · [Sponsor](https://github.com/sponsors/jeanmachuca)

## License

MIT
