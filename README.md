# service-datamonkey

REST service driving Datamonkey3

## Development

Do stuff like this:

 - `make update` to pull down the OpenAPI specification from api-datamonkey and generate a GO Gin server stub from it.
 - `make build` to build just the service-datamonkey container
 - `make start` to start the entire datamonkey 3 backend for dev/testing using docker compose
 - `make stop` to stop the datamonkey 3 backend containers


Hopefully it'll eventually have options like:
 - `make install` to manage dependencies. for now, have to manage them yourself. The important ones are golang >= 1.20 and npx w openapitools/openapi-generator-cli

**NOTE** for now you should also check out [service-slurm](https://github.com/d-callan/service-slurm) and simply `docker compose up -d` and then `docker compose down` before using this repo. I'll fix that eventually, hopefully, but its just to get built slurm images this docker compose can use.

## Testing

Make sure things are healthy:
```
export $(docker compose exec c2 scontrol token)
curl -k -vvvv -H X-SLURM-USER-TOKEN:${SLURM_JWT} -X GET 'http://localhost:9300/api/v1/health'
```

You can also do things like:

`curl -X POST -H "Content-type: multipart/form-data" -F meta='{"name":"TEST","type":"TEST TYPE","description":"TEST DESC"}' -F file=@test.txt  http://localhost:9300/api/v1/datasets` where test.txt contains anything at all. then go to `http://localhost:9300/api/v1/datasets` and confirm its worked.

Datasets uploaded will persist across re-starts of containers, etc. To clear them: `docker volume rm service-datamonkey_uploaded_data`

**PLEASE NOTE THAT CURRENTLY THIS WILL REMOVE JOB RESULTS AND LOGS AS WELL**

## DEBUGGING

To see logs for service-datamonkey: `docker logs service-datamonkey`
