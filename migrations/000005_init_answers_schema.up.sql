CREATE TABLE IF NOT EXISTS answers (
                                       id          uuid primary key default gen_random_uuid(),
                                       session_id  uuid not null references sessions(id) on delete cascade,
                                       question_id bigserial not null references questions(id),
                                       value       int not null
);