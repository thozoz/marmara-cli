# marmara-cli

Unofficial, headless command-line client for Marmara Üniversitesi's student-facing
and public APIs. Prints JSON to stdout so it's easy to script or drive from an AI
agent, and it can also run as an [MCP](https://modelcontextprotocol.io) server so
agents like Claude can call each operation as a tool.

> **Disclaimer.** This is a personal project. It is **not affiliated with,
> authorized by, or endorsed by Marmara Üniversitesi.** It talks to the same
> public/mobile endpoints the official app uses, with **your own** login
> credentials. Your password is never stored — only the returned tokens are
> cached locally in `~/.marmara/token.json` (permissions `0600`). Use at your own
> risk, and only with your own account.

## Build

Requires Go 1.24+.

```sh
go build -o marmara ./cmd/marmara     # or marmara.exe on Windows
```

## Usage

Public data needs no login:

```sh
marmara cafeteria          # daily cafeteria menu
marmara clubs              # student clubs
marmara club-events        # upcoming club events
marmara calendar           # academic calendar / exam periods
marmara campus-maps        # campus GPS markers (--types for categories)
marmara news               # university news
marmara announcements      # official announcements
marmara events             # university events
marmara features           # app feature toggles
marmara risk-report        # health/risk report
marmara directory search "yılmaz"   # personnel phone directory
marmara avesis search "yapay zeka"  # AVESİS publications/researchers
marmara ticket attributes  # support-ticket form options
```

Your own data needs a login first:

```sh
marmara login              # prompts for username/password (masked)
# or non-interactive, for scripts/agents:
MARMARA_USERNAME=... MARMARA_PASSWORD=... marmara login

marmara profile            # your profile
marmara card               # campus card balance + gate logs
marmara grades             # grade list
marmara grades detail --ders-id <OgrenciDersId>
marmara transcript
marmara schedule           # weekly timetable
marmara exams              # exam schedule

marmara logout             # revoke token + clear cache
```

Add `--pretty` to any command to indent the JSON.

The access token auto-refreshes when it's expired or near expiry, using the
cached refresh token, so you rarely need to log in again.

### Filing a support ticket

Creating a ticket is a real action against the university's live system, so it's
gated behind an explicit confirmation and is **not** exposed over MCP:

```sh
marmara ticket create --subject "..." --body "..." --yes
```

## MCP server (AI agents)

Run the binary in MCP mode over stdio:

```sh
marmara mcp
```

Register it with an MCP-aware client. For Claude Code:

```sh
claude mcp add marmara -- /full/path/to/marmara mcp
```

Every read command above is exposed as a tool (`profile`, `grades`, `schedule`,
`exams`, `cafeteria`, `calendar`, …). Log in once with `marmara login` first; the
MCP server reuses the same cached token. Ticket **creation** is intentionally
CLI-only.

## Scope

Included: the native mobile Bearer API for your own academic data (profile, card,
grades, transcript, schedule, exams) and genuinely public JSON (cafeteria,
clubs, calendar, campus maps, news, announcements, events, directory, AVESİS,
support-ticket form).

Deliberately excluded: anything that isn't a student's own data or a public
endpoint, and any cross-student lookup — calls always act as the authenticated
user and never take an arbitrary student ID.

## License

MIT — see [LICENSE](LICENSE).
