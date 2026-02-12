Just pay for Claude API. Get this working first, then experiment with cheaper APIs. If you pick GroqCloud first you run the risk of losing momentum just to save money.

This could technically be a cc plugin. The primary functionality that is missing is to reset the context after a certain point. That is already what Ralph does. But, there is a lot of other functionality that I want and I also just want to learn how an agent works underneath.

All consumer products have some sort of agent/harness. If Meta smart glasses listen to you ask a question, and then send process that audio through perhaps a local LLM into text, and then send that text to a foundation model, and then send the response back as text, and then translate that text into audio and play it for you, that is an agent. The most important software of the next decade will be agents. SY seems to think its going to be agent orchestrators, and I'm not in a position to disagree but ideologically I'd prefer we not go down the K8s path already.

# How agents work today

Start the agent
Setup a context
Add the system prompt
Add the tool instruction prompt
Take in the user input and add to context
Send the context to the LLM via the available API.

The API call will include Tools JSON to tell it what tools it can use for this request
** How does this work? Do you always send the Tools JSON

The API returns the response, which might be Tool Call JSON. Tool Call JSON can be Parallel, which contains an array of tool calls.

If it's Tool Calls, you run the tools and add the output to the context with a tool ID, so the LLM knows Tool Call created the output.

If it's just output then the LLM is likely done with the request

# Zero memory
The LLM itself has absolutely *zero* memory. You might think it does because web chat UIs and agents manage context for you, but the model at the end of the road has zero memory. It only takes in tokens and returns tokens in a stateless way. Stateless, but not pure. It's not a pure function because a given set of inputs do not always, or even normally, return the same outputs.

The ramifications of this are that the system prompt, and the tool call definitions are ALL sent EVERY time.
