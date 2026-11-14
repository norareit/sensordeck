# Development Guidelines

How work is done here is not specific to this project. It lives in AgentRite, installed as the npm package
`agentrite`, and the files [`.agentrite/instructions.md`](.agentrite/instructions.md) names are **required reading**
before any change: the kit's `process.md`, the project's standards, and the file of each role. `process.md` holds the
changes process in `changes/`, the status progression and the freeze, and the commit prefixes; the standards hold the
rules on text, documentation, commits, code and tests. The kit writes the list from `agentrite.json`. This file holds
only what is specific to this project, and nothing is stated in both places.

Read [README.md](README.md) before making changes. It is the source of truth for the project itself.

The kit is a dependency: `npm ci` installs it, on the host or inside the sandbox, and moving to a newer release is a
change to `package.json` and `package-lock.json`. An improvement to it is made in the kit repository, never under
`node_modules/`.

## Roles

[`.agentrite/instructions.md`](.agentrite/instructions.md) names the file of each of the four roles that do the work
in `changes/`. The harnesses reach them through the role wrappers in these directories. Each wrapper is a pointer to
one role file and holds no instructions of its own:

* `.claude/agents/`
* `.opencode/agents/`

## Changes in this project

The change check is `npm run check-changes`, which runs the kit's script.

## Specific to this project

<!--
What is specific to this project goes here, under headings of its own: for example its house style, the rules its
code follows, and how its tests are run. Everything that holds for every project is in the files
.agentrite/instructions.md names already.
-->

### Publishing rules

* The project is MIT-licensed. Write all code and docs independently. Never translate or copy GPL-licensed code
* **Never commit vendor or third-party `.dat` themes.** Keep them in `local-themes/` (gitignored), for local testing
  only

### The automated build of the initial plan

The project is built from its initial plan by an automated loop, in which an orchestrating agent session on the host
stands in for the human at the steps `process.md` reserves for the human. The human delegated them for this project.
The loop writes one unit at a time, and the next only once the one before it is merged.

* The orchestrating session reads each spec, or the record of a task, and either has it corrected or approves it with
  `npx agentrite approve`. It decides which items of "Beyond the request" are taken. Before the approval it logs in
  `record.md` what it checked and decided, so that the `[human]` entry of the approval is read as one made under this
  delegation
* It decides the blocking findings and the proposed declines a run of the orchestrator stops for, and logs each
  decision in `record.md`
* After a run that ended with no open blocking finding, it sets the unit to `implemented` and merges the unit's branch
  into the default branch with `git merge --ff-only`
* A check that needs the physical device is the human's, and so is everything on Windows. The session asks for it and
  waits, and does not take a guess for a result
* The delegation holds for this loop only. A session that is not running it follows `process.md` as it stands
