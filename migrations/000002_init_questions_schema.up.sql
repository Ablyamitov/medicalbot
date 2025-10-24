CREATE TABLE IF NOT EXISTS questions (
                           id          bigserial primary key,
                           test_id     bigserial not null references tests(id) on delete cascade,
                           question_no int  not null, -- порядковый номер вопроса
                           text        text not null
);