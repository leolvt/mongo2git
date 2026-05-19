# mongo2git

Export a MongoDB collection to git-backed JSON files — perfect for version-controlled backups.

Each document is written as a human-readable [MongoDB Extended JSON](https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/) file, committed, and pushed to a git repository.

## Quick start

```bash
# Clone and build
git clone https://github.com/leolvt/mongo2git.git
cd mongo2git
go build -o bin/mongo2git ./cmd/mongo2git

# Run (all options can also be set via environment variables — see .env.example)
./bin/mongo2git \
  --mongo-uri   mongodb://localhost:27017/mydb \
  --mongo-collection mycollection \
  --dump-dir    data \
  --clone-dir   /tmp/backups \
  --repo-url    git@github.com:user/backups.git \
  --local
```

## How it works

```
MongoDB  ──fetch──▶  JSON files  ──commit──▶  Git repo  ──push──▶  Remote
                         │                              │
                         └── one .json per document     └── Slack notification (optional)
```

1. Connect to MongoDB and fetch all documents from the collection
2. Write each document as a `.json` file (MongoDB Extended JSON, mongoimport‑ready)
3. Clone (or fetch+reset) a git repository
4. Stage, commit, and push all changes
5. Notify via Slack or stdout

## Usage

```
Usage: mongo2git [options]

Required:
  --mongo-uri, -m         MongoDB connection URI (or MONGO_URI env)
  --mongo-collection, -c  MongoDB collection to dump (or MONGO_COLLECTION env)
  --dump-dir, -d          Subdirectory for dumped JSON files (or DUMP_DIR env)
  --clone-dir, -g         Local directory for cloned git repo (or CLONE_DIR env)
  --repo-url, -r          Git repo SSH URL (or REPO_URL env)

Optional:
  --git-branch, -b        Git branch to push to (or GIT_BRANCH env, default: main)
  --slack-webhook-url, -s Slack incoming webhook URL (or SLACK_WEBHOOK_URL env)
  --local, -L             Commit only, no push; log notifications instead of Slack
  --version, -v           Print version and exit
```

## Environment variables

All options can be set via environment variables. See [`.env.example`](.env.example) for a complete template.

| Variable | Required | Default | Description |
|---|---|---|---|
| `MONGO_URI` | yes | — | MongoDB connection string with database name |
| `MONGO_COLLECTION` | yes | — | Collection to export |
| `DUMP_DIR` | yes | — | Subdirectory inside clone for JSON files |
| `CLONE_DIR` | yes | — | Local path where the git repo is cloned |
| `REPO_URL` | yes | — | Git remote URL (SSH recommended) |
| `GIT_BRANCH` | no | `main` | Branch to push to |
| `SLACK_WEBHOOK_URL` | no | — | Slack incoming webhook for notifications |

## Output format

Documents are serialized as [MongoDB Extended JSON](https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/) (relaxed mode) — directly importable with `mongoimport`, MongoDB Compass, or `mongosh`:

```json
{
  "_id": { "$oid": "507f1f77bcf86cd799439011" },
  "name": "example",
  "created": { "$date": "2025-01-15T10:30:00Z" }
}
```

Each document is saved as `<dump-dir>/<_id>.json`. Supported `_id` types: ObjectID, UUID (binary subtype 4), string, and generic binary.

## Running in CI / cron

For automated backups, set the required environment variables and run with `--local` to skip the Slack notification:

```bash
export MONGO_URI="mongodb://localhost:27017/mydb"
export MONGO_COLLECTION="users"
export DUMP_DIR="backups/users"
export CLONE_DIR="/data/backups"
export REPO_URL="git@github.com:org/backups.git"
export GIT_BRANCH="auto"

mongo2git --local
```

With Slack notifications:

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
mongo2git
```

## Requirements

- [Go](https://go.dev/) 1.26+
- [git](https://git-scm.com/) installed and on `$PATH`
- SSH access to the git remote (with host key pre‑configured in `~/.ssh/known_hosts`)
- MongoDB 4.0+ (uses the v2 Go driver)

## License

MIT — see [LICENSE](LICENSE).

---

*This project was developed with the assistance of [Pi](https://pi.dev), an AI coding agent.*
