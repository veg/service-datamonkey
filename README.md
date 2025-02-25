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

Starting jobs I'd recommend using Postman, for convenience. Whatever method though, you want to use a url like `http://localhost:9300/api/v1/methods/fel-start` to start and monitor jobs, and one like `http://localhost:9300/api/v1/methods/fel-result` to get results. POST body should look something like:
```
{
  "alignment": "2ddaaa7f2d54e25f81062ab8cda13b38",
  "tree": "31fa9ce04076f0f9dc403278c7c1717c",
  "ci": false,
  "srv": true,
  "resample": 0,
  "multiple_hits": "None",
  "site_multihit": "Estimate",
  "genetic_code": {
    "value": "Universal",
    "display_name": "Universal code"
  },
  "branches": []
}
```

Here, `alignment` and `tree` are references to dataset ids of uploaded data (see below). Starting, monitoring and fetching results for methods also requires the `X-SLURM_USER_TOKEN` header in the request, similar to the health endpoint (see above).

Make sure jobs tracker from previous dev session is cleared: `docker compose exec c2 rm /data/uploads/job_tracker.tab`. (Sorry, I'll clean that up as well in a bit. Maybe add something to the Makefile..)

You can also do things like:

`curl -X POST -H "Content-type: multipart/form-data" -F meta='{"name":"TEST","type":"TEST TYPE","description":"TEST DESC"}' -F file=@test.txt  http://localhost:9300/api/v1/datasets` where test.txt contains anything at all. then go to `http://localhost:9300/api/v1/datasets` and confirm its worked.

Datasets uploaded will persist across re-starts of containers, etc. To clear them: `docker volume rm service-datamonkey_uploaded_data`.

**PLEASE NOTE THAT CURRENTLY THIS WILL REMOVE JOB RESULTS AND LOGS AS WELL**

If instead you'd like to remove specific files: `docker compose exec c2 rm /data/uploads/[filename]`

## DEBUGGING

To see logs for service-datamonkey: `docker logs service-datamonkey`
