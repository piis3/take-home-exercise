# Take home exercise

## General approach
This stores the list of exit nodes as pulled from `www.dan.me.uk` in a table in a postgres database. 
The data is stored for each time we fetch the data with the timestamp so that we can ask for what nodes were available at specific times or over date ranges. 

A fetch history is stored in the `node_fetch_history` table. This table is used for making sure we don't query from the provider too frequently, as they have pretty aggressive rate limits. 
It's also used with a `select for update` operation to ensure that we don't have multiple fetches running concurrently. 
The whole process of fetching the node list and inserting it into the table is done within a transaction in order to preserve the lock's integrity and to avoid partial results being made available to callers. 
Long running transactions with outside IO happening are usually something to be careful about but since we've ensured that there can be only one of these at a time, that seems acceptable. 

The fetches are driven from a simple go timer, if there are multiple instances of this process running, that's fine because they'll coordinate through the lock on the `node_fetch_history` table.

The node list is made available through a simple rest like API that supports pagination. 

## API
- `GET /nodes?limit=N&token=sometoken` provides data from the most recent data fetch. The `token` parameter comes from `PagingToken` of a previous response or can be left blank. Limit is also optional.
- `GET /nodes/time-range?limit=N&token=sometoken&start_ts=2024-08-07T16:00:00Z&end_ts=2024-08-07T19:00:00Z` provides all exit nodes seen during a time range along with the timestamp of the most recent dataset they were seen in. `token` and `limit` are the same as in the `/nodes` call. `start_ts` and `end_ts` are RFC3339 timestamps with an optional timezone. If omitted, the last week of data will be provided.
- `GET /allow-list?limit=N&token=sometoken` provides the current allow-list, which is a list of addresses that will be omitted from the node results above. `token` and `limit` are the same as above.
- `PUT /allow-list/<address>` will add the provided address to the allow-list. No request body is necessary.
- `DELETE /allow-list/<address>` will remove the provided address from the allow-list.

## Implementation
I used a library called `jet` for building type-safe queries to the DB. This scans the database schema at build time and generates code and model objects for interacting with the database. This turned out pretty well but it lacked a couple of surprising features, such as being able to use a `where not exists` clause. SQL is pretty flexible so I worked around this with an outer join.

For managing the schema, I'm using the `migrate` library which keeps track of schema state and can automatically apply scripts in the right order. This could also be done by the code during service startup in order to manage the schema in a production database. 

Postgres can be started with the docker-compose file. I also included a web server to hold some canned data to develop against because of the rate limits on the `www.dan.me.uk` service.

## Building
Building is pretty simple, start docker compose to get the database. Since schema inspection is used at build time, we need a postgres DB running to build. Then just `make`. 