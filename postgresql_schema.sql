create table group_members(id BIGSERIAL PRIMARY KEY, group_id bigint not null, user_id bigint not null, nickname varchar(300) not null, rights integer not null default(0));
create table groups(id BIGSERIAL PRIMARY KEY, group_number bigint not null, name varchar(300) not null);
create table users(id BIGSERIAL PRIMARY KEY, qq_number bigint not null, qq_name varchar(300) not null);
create table replies(id BIGSERIAL PRIMARY KEY,author_id bigint not null, key varchar(30) not null, reply varchar(1000) not null, group_id bigint not null);
create table discuss_groups(id BIGSERIAL PRIMARY KEY,discussion_num bigint not null, user_id bigint not null);
create table discuss_replies(id BIGSERIAL PRIMARY KEY,author_id bigint not null, key varchar(30) not null, reply varchar(1000) not null, discussion_num bigint not null);