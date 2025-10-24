CREATE TABLE IF NOT EXISTS sessions (
                                        id          uuid primary key default gen_random_uuid(),
                                        patient_id  varchar(255) not null,
                                        test_id     bigserial not null references tests(id),
                                        current_step int not null default 1,
                                        status      varchar(50) not null default 'started',
                                        test_type   varchar(50),
                                        created_at  timestamp not null default now()
);