# oauth-tester
i need to learn more about oauth/oidc. i have a self-hosted enterprise product. it can connect to entra ID and GitHub etc etc like any other platform. its auth systems are not special or unique in any way. but, i don't want to muck around in the cloud to get my lab environment integrated with an auth provider. i want to write a very small provider that i can plugin to.

the docs for the product i am going to integrate with are here
https://www.astronomer.io/docs/astro-private-cloud/v-1-x/integrate-auth-system

oauth-tester is IdP that implements OIDC.





# Workflow
I will provide a Dicovery URL to my enterprise product as part of configuring this OpenID Provider.

There are only two fields that I have access to in the enterprise product: discoveryUrl and clientId. That's all I need from oauth-tester.

I think these are the endpoints I need

* Expose a Discovery doc: GET /.well-known/openid-configuration

* Expose a JWKS doc: GET /jwks

* An Auth page: GET /auth (where you just click "Login" and it redirects back).

* A Token endpoint: POST /token (where you hand out the JWTs).

# UI/UX
oauth-tester will have a web UI

First, the web UI will provide the normal login page that the enterprise app user expects. The same login page that all IdPs provide.

Second, the web UI will provide a simple page that lets you add, delete, or edit a user. Perhaps its just a grid that lets the user edit one cell at a time and then just updates the DB every time.


## oauth-tester shape
oauth-tester is a single Golang binary. It has no web UI. All configuration is provided by environment variables.


## Language and frameworks
Pure Go.

sqlite is handled by modernc, pure Go
https://gitlab.com/cznic/sqlite





## Security
Security is not a priority for this tool. As long as I can login correctly from the enterprise product then I am happy.

## Testing
I can't think of any way for you to test anything. You are free to write lots of unit tests but I'll have to do the integration testing myself. Try and be as confident as you can about the implementation.

## Features that won't be implemented
* Security is not important
* custom oauth flows

## SCIM
SCIM is required but not initially. Make sure to architect the code such that SCIM can be added when I am ready for that.

## State
All state is stored in sqlite



## Open Questions

