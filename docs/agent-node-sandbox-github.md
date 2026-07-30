# Agent Node Sandbox + GitHub Repository

This scaffold can optionally mount one GitHub repository into the support agent node. Leave the sandbox env vars unset for the default WhatsApp + Slack support agent.

## Enable It

Add these values to `.env.local`:

```bash
AGENT_SANDBOX_GITHUB_REPO_URL=https://github.com/example-org/example-public-repo
AGENT_SANDBOX_GITHUB_REPO_BRANCH=main
AGENT_SANDBOX_GITHUB_PAT=
AGENT_SANDBOX_NETWORK_MODE=allow_list
AGENT_SANDBOX_ALLOWED_OUTBOUND_HOSTS=docs.example.com,api.example.com
```

Public repositories can leave `AGENT_SANDBOX_GITHUB_PAT` empty. Private repositories should set it to a GitHub token with read access to the repository.

`AGENT_SANDBOX_ENABLED` is optional. If a repo URL is present, `workflow.ts` enables the sandbox by default when the CLI compiles the workflow. Set `AGENT_SANDBOX_ENABLED=false` only when you want to keep the repo values in `.env.local` but deploy the workflow with the sandbox disabled.

## What The Workflow Source Adds

`workflows/whatsapp-support-agent/workflow.ts` reads these env vars when the CLI compiles the workflow and includes the sandbox fields in the agent node config:

```json
{
  "sandbox_enabled": true,
  "sandbox_network_mode": "allow_list",
  "sandbox_allowed_outbound_hosts": ["docs.example.com", "api.example.com"],
  "flow_agent_resources": [
    {
      "resource_type": "github_repository",
      "repo_url": "https://github.com/example-org/example-public-repo",
      "branch": "main"
    }
  ]
}
```

The same shape is available in `config/examples/agent-node-sandbox-github.json`.

## Mounted Path

Inside the agent sandbox, repositories are mounted under `/workspace/repos`.

For branch `main`:

```text
/workspace/repos/example-org-example-public-repo
```

For a non-main branch such as `feature/support-handoff`:

```text
/workspace/repos/example-org-example-public-repo@feature--support-handoff
```

When the repository sandbox is enabled, the workflow source also appends prompt guidance telling the support agent to inspect that mounted repo before escalating repository-specific or code-specific questions to Slack.

## Network Mode

Use `AGENT_SANDBOX_NETWORK_MODE=allow_list` by default. Add any extra outbound hosts the agent needs in `AGENT_SANDBOX_ALLOWED_OUTBOUND_HOSTS`.

Use `AGENT_SANDBOX_NETWORK_MODE=allow_all` only for examples where unrestricted outbound network access is intentional.
