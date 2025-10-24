create table tests (
                       id          bigserial primary key,
                       type        varchar(50) not null unique, -- AMS, MIEF, IPSS
                       name        varchar(255) not null,
                       description text,
                       disclaimer  text
);