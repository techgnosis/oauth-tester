# oauth-tester
i need to learn more about oauth/oidc. i have a self-hosted enterprise product that I will refer to as APC. APC can connect to entra ID and GitHub etc etc like any other platform. APCs auth systems are not special or unique in any way. but, i don't want to muck around in the cloud to get my lab environment integrated with an auth provider. i want to write a very small provider that focuses on ease of use and transparency

the docs for APC are here
https://www.astronomer.io/docs/astro-private-cloud/v-1-x/integrate-auth-system

oauth-tester is IdP that implements OIDC.



# Workflow
I will provide a Dicovery URL to APC as part of configuring this OpenID Provider.

There are only two fields that I have access to APC: discoveryUrl and clientId. That's all I need from oauth-tester.

I think these are the endpoints I need

* Expose a Discovery doc: GET /.well-known/openid-configuration

* Expose a JWKS doc: GET /jwks

* An Auth page: GET /auth (where you just click "Login" and it redirects back).

* A Token endpoint: POST /token (where you hand out the JWTs).

# UI/UX
oauth-tester will have a web UI

First, the web UI will provide the normal login page that the APC user expects. The same login page that all IdPs provide.

Second, the web UI will provide a simple page that lets you add, delete, or edit a user. Perhaps its just a grid that lets the user edit one cell at a time and then just updates the DB every time. Passwords are stored and displayed in plain text.

The UI should have a page called "logs" where every OIDC-related request and response is logged. It's crucial that I can see how everything works at the HTTP level.


## oauth-tester shape
oauth-tester is a single Golang binary. oauth-tester should always be a static compile.


## Language and frameworks
Pure Go.

sqlite is handled by modernc, pure Go
https://gitlab.com/cznic/sqlite

## TLS
oauth-tester should serve everything over TLS. I will provide a cert and key file on disk called cert.pem and key.pem



## Security
Security is not a priority for this tool. As long as I can login correctly from APC then I am happy.

## Testing
You can and should write as many unit tests as you want.

I should be able to open oauth-tester and CRUD users via the UI

I'll have to integrate oauth-tester into my cluster to truly test it but that's not your responsibility

## Ports
I don't know enough about the subject but I think oauth-tester only needs to listen on 443

## Features that won't be implemented
* Security is not important
* custom oauth flows

## SCIM
SCIM is required but not initially. Make sure to architect the code such that SCIM can be added when I am ready for that.

## State
All state is stored in sqlite



## Open Questions
* Is 443 the only port that oauth-tester needs? Are there special OIDC ports?
