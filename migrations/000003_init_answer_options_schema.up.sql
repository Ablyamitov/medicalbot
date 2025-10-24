CREATE TABLE IF NOT EXISTS answer_options (
                                      id          uuid primary key default gen_random_uuid(),
                                      question_id bigserial not null references questions(id) on delete cascade,
                                      value       int not null,
                                      text        text not null
);