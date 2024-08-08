create table if not exists allow_list(
  node_address varchar(255) primary key,
  created_at timestamp not null default currenttime()
);