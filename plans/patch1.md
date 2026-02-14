# Machine readable logs

Right now I can access the OIDC exchange logs at /ui/logs. Please add an endpoint so that the logs can be fetched in a machine readable format so that you can query it yourself. Lets make it so that you can request an HTTP verb and an endpoint and the number of results you want. For example, if you wanted to see if a POST to /token was working, you would request POST, /token, and 3. That would show the 3 latest attempts at POST to /token.

I suspect you are storing all these logs in the DB, so perhaps you could specify the number of minutes in the past for fetching the logs? I'm open to ideas.

It could be JSON but ideally it's a more LLM friendly format. I'm thinking minified JSON since the payloads are complex and it might be too much for YAML.

There is a "CLear Logs" button in the UI right now. That might be good for you to be able to use. That way you can clear the logs after you gather what you want and then I can use the app again and you can gather whatever you need.

Your environment has curl and jq already but I suspect jq is not needed for you, just for humans.