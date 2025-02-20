# service-datamonkey
REST service driving Datamonkey3

Do stuff like this:

`make update` to pull down the OpenAPI specification from api-datamonkey and generate a GO Gin server stub from it.

It'll eventually have options like:
`make install` to manage dependencies. for now, have to manage them yourself.
`make build` to build just the service-datamonkey container
`make start` to start the entire datamonkey 3 backend for dev/testing using docker compose
`make stop` to stop the datamonkey 3 backend containers
