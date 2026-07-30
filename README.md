# WhatsApp Support Agent

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Built with Kapso](https://img.shields.io/badge/Built_with-Kapso-black.svg)](https://kapso.ai)

Open-source Kapso example repo for a WhatsApp support workflow that escalates uncertain questions to Slack, pauses with `enter_waiting`, and resumes the same agent run when the team answers.

```
WhatsApp customer
  asks a question
      |
      v
Kapso workflow agent
  answers if confident
  calls ask_team_question if uncertain
      |
      v
Slack support thread
  team replies and sends "done"
      |
      v
slack_events function
  aggregates replies and resumes the workflow
```

## What It Contains

| Resource | Path | Description |
|---|---|---|
| `whatsapp-support-agent-ask-team-question` | `functions/whatsapp-support-agent-ask-team-question/` | Private Kapso function used by the `ask_team_question` agent tool |
| `whatsapp-support-agent-slack-events` | `functions/whatsapp-support-agent-slack-events/` | Public Kapso function that receives Slack events and resumes the workflow |
| `whats-app-support-agent-example` | `workflows/whats-app-support-agent-example/workflow.ts` | Workflow source authored with `@kapso/workflows` |

The normal loop is: edit function files, edit `workflow.ts`, run `kapso push`.

`kapso push` compiles `workflow.ts` by reading its default-exported `Workflow` instance, writes `workflow.yaml` and `definition.json`, then pushes the functions, workflow, and trigger. This scaffold ignores those generated workflow files because `workflow.ts` is the source of truth. You can track generated JSON/YAML instead if you prefer CLI-only editing.

## Prerequisites

- Bun `1.3+`
- Kapso CLI on your `PATH`
- A Kapso project and `KAPSO_API_KEY`
- A WhatsApp phone number id for inbound triggers
- A Slack app with `chat:write`, `channels:history`, a signing secret, and a public channel where the bot is invited

## Setup

```bash
bun install
cp .env.example .env.local
```

For this unpublished local checkout, `bun install` creates `node_modules/.bin/kapso`. Add it to your shell path when you want to run bare CLI commands from this repo:

```bash
export PATH="$PWD/node_modules/.bin:$PATH"
```

After the CLI is published, install it globally instead and keep using the same commands:

```bash
bun add -g @kapso/cli
```

Fill `.env.local` with:

| Variable | Required | Description |
|---|---|---|
| `KAPSO_API_KEY` | Yes | Project-scoped Kapso API key |
| `WHATSAPP_PHONE_NUMBER_ID` | Yes | Phone number for inbound triggers |
| `SLACK_BOT_TOKEN` | Yes | Slack bot OAuth token |
| `SLACK_SIGNING_SECRET` | Yes | Slack app signing secret |
| `SLACK_CHANNEL_ID` | Yes | Channel where escalations are posted |
| `PROVIDER_MODEL_NAME` | No | Defaults to `openai/gpt-5.4` |

Validate locally:

```bash
bun run validate
bun test
```

Link and push:

```bash
kapso link
kapso push
bun run sync:secrets
```

After `sync:secrets`, copy the printed `Slack events URL` into the Slack app Event Subscriptions page. The setup is not complete until Slack points at that URL and the bot is invited to `SLACK_CHANNEL_ID`.

## Commands

| Command | What it does |
|---|---|
| `kapso link` | Binds this directory to the Kapso project |
| `kapso push --dry-run` | Shows the function/workflow push plan |
| `kapso push` | Builds workflow source and pushes functions, workflow, and trigger |
| `bun run sync:secrets` | Pushes per-function secrets from `.env.local` |
| `bun run validate` | Checks local function sources and workflow source |
| `bun test` | Runs unit tests |

## Advanced: Agent Sandbox

This scaffold can optionally mount one GitHub repository into the agent node sandbox so the support agent can inspect product docs, code, or examples before escalating to Slack.

This is not special CLI behavior. The CLI only compiles `workflow.ts`; this scaffold's workflow source reads `AGENT_SANDBOX_*` env vars and conditionally adds agent-node config through the workflow library's `rawConfig`.

Set these in `.env.local` before `kapso push`:

- `AGENT_SANDBOX_GITHUB_REPO_URL`
- `AGENT_SANDBOX_GITHUB_REPO_BRANCH`, defaults to `main`
- `AGENT_SANDBOX_GITHUB_PAT`, empty is valid for public repos
- `AGENT_SANDBOX_NETWORK_MODE`, defaults to `allow_list`
- `AGENT_SANDBOX_ALLOWED_OUTBOUND_HOSTS`, comma-separated or newline-separated
- `AGENT_SANDBOX_ENABLED`, optional explicit `true` or `false`

Mounted paths are predictable:

- `main`: `/workspace/repos/<owner>-<repo>`
- non-main branch: `/workspace/repos/<owner>-<repo>@<branch-slug>`

See [docs/agent-node-sandbox-github.md](docs/agent-node-sandbox-github.md) for details.

## Function Source Contract

Kapso function entrypoints are plain uploaded Worker files:

```js
async function handler(request, env) {
  return new Response('ok');
}
```

They intentionally do not use `export default`, `module.exports`, or a TypeScript build step.

## Repo Layout

```
functions/                  # Kapso function source files and function.yaml metadata
workflows/                  # Workflow-as-code source
scripts/sync-secrets.js     # Temporary helper until function secrets live in the CLI
src/lib/                    # Shared local helpers for workflow source and validation
tests/                      # Unit tests for function behavior and workflow source
```

## Notes

- The Kapso API base URL is hardcoded to `https://api.kapso.ai`.
- Function secrets are still synced separately because the CLI does not manage per-function secrets yet.
- `@kapso/cli` and `@kapso/workflows` are regular package dependencies. Use local links only when testing unpublished package changes.

## License

[MIT](LICENSE)
