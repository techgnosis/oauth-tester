# bdagent
A coding agent that integrates deeply with `bd` to allow for 24/7 autonomy.


# Workflow
Like any other agent, the first task is to tell the LLM to read your spec and fill bd with epics and issues.

Ideally an epic is something that the LLM would work on for ~20 minutes.

Next, the LLM is instructed to start working on the next epic in bd and to stop working at the end of that epic.

Then, we instruct the LLM to land the plane.

Then the context is wiped and we start anew, with the LLM working on the first epic.


## bdagent shape
bdagent is a CLI. It is not a TUI. It does not have panes or fixed UI of any kind. It does not allow for interactive chat-based engineering. The only user input it collects is in Review mode where it asks the user questions about the issues in `bd` and then refines the contents of `bd`.

## Tools
* Read
* Edit
* Write
* bash

The only open question is whether `bd` gets its own tool definition. Based on my usage of CC and Opus, I suspect its not needed. It's already very good at using `bd`.

## Language and frameworks
Pure Go. No frameworks that I am aware of that I would need. No TUI, stdlib has HTTP and user input.

## Input
No default system prompt - must be provided or agent won’t start. Must be provided in a text file.

No default tool prompt - must be provided or agent won't start. Must be provided in a text file.

Mode flag:
* Plan - You provide your written spec in a single file. The LLM is given a pre-defined prompt to read the spec and break it down into bd epics and issues.
* Review - the LLM is given a pre-defined prompt to read all of bd, not just open issues, and ask the user questions about what to put into bd. The user can and should run Review mode multiple times
* Implement - the model is given a pre-defined prompt to start implementing the project by starting with the first bd epic available.

## Review mode
Review mode is the only interactive mode.

* display those questions one at a time
* give the answers back to the LLM
* The LLM returns back tool calls to add its new opinion of the bd issues into bd

## Output
The LLM output is streamed, response by response, and each response shows the number of tokens that went into it and the number of tokens that it returned.

Log all input, output and tool use in a structured format like JSONL
Have the agent have an audit mode that reads the JSONL log and prints meaningful helpful output

## Context and tokens
bdagent prioritizes transparency in token usage and context size. Context size is always shown prominently, as well as the input and output tokens of every request and response.

Total size of context is shown at all times in the UI. I'm not 100% sure how to measure the context in tokens. I suppose I have to send the context to the LLM first and then read the response which will include something like `input_tokens`

Input tokens and output tokens are shown at all times in the UI

## Security
bdagent always has full permissions. It is built to be run in a container. Users, myself included, don't trust built-in sandboxing features anway.

bdagent should output something like "bdagent has no sandboxing. It has full permission to do whatever it wants. bdagent is meant to be run in a container. Only proceed if you understand this"

# Testing
This part is very important.

## Features that won't be implemented
* Sub-agents
* MCP
* Multi-model sessions. Only one model at a time.
* Compaction


bdagent does not create ANY state on disk outside of bd:
* No Projects
* No Plans
* No TODOs
* Nothing in the user's home directory
* No OAuth session data, only uses API tokens from environment variables



## Open Questions
* In Review mode, should bdagent read bd for the model? Or should bdagent prompt the model to read bd. The agent could save tokens by reading bd for the model
* Does bdagent monitor the epic that is being worked on to ensure that the agent stops after the epic is done? Right now the assumption is that the model is prompted to say "I'm done" when the epic is done and it's landed the plane. But should bdagent take a more active role?
* Is "one epic" the right lifecycle? Maybe bdagent monitors context size instead? How much context is too much? What if bdagent should clear the context after ever single issue? Is that better for token cost?
* How does "Thinking" work? When Claude says stuff to signify it's thinking, does that count as output tokens? I only want to show data in the UI that affects token usage.
* Does bd gets its own tool definition or do I rely on bash? Claude Code does not send Opus a bd tool definition but Opus is already incredibly good at using bd. Evidence suggests it doesn't need one.
