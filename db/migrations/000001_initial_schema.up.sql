create table if not exists node_fetch_history(
  id serial primary key,
  fetch_time timestamp unique not null
);

create table if not exists exit_nodes (
  node_address inet not null,
  fetch_time timestamp not null,
  primary key (node_address, fetch_time)
);

create index if not exists exit_node_fetch_time on exit_nodes (fetch_time);