# Prompt composition: strategies, contexts, patterns, and sessions

This document explains how Fabric builds the prompt that is sent to the model when you use the CLI flags (or API fields) `--strategy`, `--context`, `--pattern`, and `--session`. It shows the exact order of composition, how the flags interact, what gets stored in sessions, and practical recommendations so you can get predictable, repeatable results from Fabric.

> Short summary: When all flags are used, Fabric constructs a system message by combining strategy → context → pattern (plus optional language instructions), then sends that system message together with the session history and the current user message to the model. In raw mode the system/pattern may be merged into the user message instead.

Contents
- What each flag is for
- Where strategy/context/pattern files live
- Exact order of prompt construction (including language handling)
- Full message sequence sent to the model (new and existing sessions)
- Raw mode special case
- Best practices and suggested workflows
- Troubleshooting and pointers

---

## What each flag/field does

- `--strategy <name>` (request.strategyName)
  - Selects a strategy JSON file (by filename) whose `prompt` text is prepended to the system prompt. Strategies are intended as small reasoning tactics or prompt transformations (e.g., "Chain of Thought", "Red Team"). The strategy file can include `name`, `description`, and `prompt`.

- `--context <name>` (request.contextName)
  - Selects a named context (persistent snippet of text) that is injected into the system prompt. Use contexts for reusable system-level instructions or domain knowledge you want to apply repeatedly (e.g., company style guides, domain constraints).

- `--pattern <name>` (request.patternName)
  - Selects a pattern from Fabric's pattern store. Patterns are the main templates that define the specific task (e.g., summarization, code generation). Pattern variables can be applied when you use `-v/--variable`.

- `--session <name>` (request.sessionName)
  - Loads a named session containing prior messages, appends new messages to it, and saves back to disk. Sessions are used to maintain conversational state across invocations.

---

## Where files live

- Strategies: `~/.config/fabric/strategies` (JSON files). Default repo copy is at `data/strategies` in the project.
- Contexts: `~/.config/fabric/contexts`
- Patterns: `~/.config/fabric/patterns` (or the repo `data/patterns`)
- Sessions: `~/.config/fabric/sessions` (sessions are saved as JSON)

Use `fabric -S` (setup) to download/populate strategies, and the CLI/REST endpoints to list/manage contexts and sessions.

---

## Exact order of system prompt construction

When Fabric builds a chat request it composes the system message in the following order (top → bottom):

1. strategy.Prompt (if `--strategy` is set and the strategy JSON contains `prompt`)  
   (strategy is prepended to the rest)

2. context content (if `--context` is set)

3. pattern content (if `--pattern` is set)

4. language instruction (if a non-default language is set, e.g. `--language=fr`)  
   - Fabric appends an instruction such as:
     IMPORTANT: First, execute the instructions provided in this prompt using the user's input. Second, ensure your entire final response ... is written ONLY in the <language>.

In code-speak:
- patternContent and contextContent are combined first:
  systemMessage := Trim(contextContent) + Trim(patternContent)
- then strategy is prepended:
  systemMessage = strategy.Prompt + "\n" + systemMessage
- then language instruction may be appended

Note: If neither context nor pattern nor strategy is present, no system message is added.

---

## Full message sequence actually sent to the model

Fabric composes a list of messages (roles: system, user, assistant) to send to the vendor. There are two main cases:

A) New session (no previously saved messages)
- system message: the combined systemMessage described above (strategy → context → pattern → language instruction) — role `system`
- user message: the immediate user input (after template/variable substitution if applicable) — role `user`

B) Existing session (session contains prior messages)
- all previous vendor messages from the session (system/user/assistant messages in chronological order)
- then append the current system message (strategy → context → pattern → language instruction)
- then append the new user message

Important detail:
- Fabric stores and reuses the full conversation history in the session file; when you attach to a session, the entire prior conversation will be part of the vendor messages list sent to the model, followed by the new system message and user message (see BuildSession behaviour).

Meta messages:
- Fabric may append a `meta` role message into the session, but `meta` messages are filtered out of the vendor messages and are not sent to the model.

---

## Raw mode special-case

When the provider or model requires "raw" mode (or you explicitly request raw), Fabric may combine the systemMessage and user content into a single user message instead of sending a separate system message. Behavior:

- If `raw` is true and `pattern` is set, Fabric sets the final user content to the systemMessage (pattern is used directly).
- If `raw` is true and no pattern is set, Fabric concatenates systemMessage and the user content into a single user message:
  finalContent := systemMessage + "\n\n" + userContent
- Attachments (multi-part content) are preserved; Fabric will build a MultiContent message with the combined text and non-text parts.

So in raw mode the model may not receive a separate `system` role message — instead the instructions are embedded in a `user` message. This can change how the model treats those instructions.

---

## Example

Given:
- `--strategy=chain_of_thought` with prompt: "Think step-by-step and show your reasoning."
- `--context=writer` with content: "Use formal tone and follow company style."
- `--pattern=summary_short` with content: "Summarize the following text in 3 bullets."
- User input: "Here is the document: <doc text>"
- `--session=mychat` (existing session may have prior messages)

Final system message (combined):
Think step-by-step and show your reasoning.
Use formal tone and follow company style.
Summarize the following text in 3 bullets.

(If `--language=fr` is set, an appended translation instruction would follow.)

If `mychat` has prior messages, those prior messages are included first in the sequence sent to the model. If not, the model receives the system message above, followed by the user message containing the document text.

---

## Best practices and recommendations

- Use contexts for stable, organizational or domain-level constraints (tone, brand, long domain facts). Contexts are persistent, reusable, and intended to be combined with patterns.

- Use patterns for the primary task template—what you want the model to do (summarize, rewrite, code-gen). Patterns can accept variables (use `-v` / `--variable`).

- Use strategies for temporary reasoning or tactic-level instructions (e.g., "think step-by-step", "red-team this answer", "use an iterative draft process"). Strategies are short and intended to be easily swapped (you can list and install them).

- Order matters: strategy + context + pattern. If you want a strategy to override or modify the system behavior, ensure its prompt is written accordingly and test with `--liststrategies` and example runs.

- If you need the strategy to act after the context or pattern (rare), consider writing an equivalent strategy that targets the desired position, but be mindful that code always prepends strategy text.

- When running models that require the system role separate (OpenAI-like), avoid raw mode unless you intentionally want to embed system instructions inside the user message.

- For reproducible results, use sessions when you want to retain conversation context, and wipe sessions when you want a fresh start (`--wipesession`).

---

## Troubleshooting

- Strategy not applied?
  - Ensure the strategy JSON file exists in `~/.config/fabric/strategies` and has a `prompt` field. Filename can be provided without `.json`.
  - Use `--liststrategies` to see installed strategies.

- Context not applied?
  - Check `~/.config/fabric/contexts/<name>` and verify content. Use `--printcontext <name>`.

- Pattern variables not replaced?
  - If you passed `--no-variable-replacement`, variable substitution will be skipped.
  - Use `-v/--variable` to provide variables, or `--input-has-vars` if your user input contains template variables.

- Unexpected translations or language behaviour?
  - If `--language` is set to a non-default, Fabric appends a language instruction to the system message — this can alter the model's output. Remove or change `--language` if not desired.

---

## Related commands / endpoints

- CLI:
  - `fabric -S` — setup/install strategies
  - `fabric --liststrategies`
  - `fabric --listcontexts`, `--printcontext`, `--wipecontext`
  - `fabric --listsessions`, `--printsession`, `--wipesession`

- REST API:
  - `/contexts/*` endpoints for contexts
  - `/sessions/*` endpoints for sessions
  - `/strategies` endpoint returns available strategies metadata (name, description, prompt)
