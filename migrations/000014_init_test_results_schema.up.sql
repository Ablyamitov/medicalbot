CREATE TABLE IF NOT EXISTS test_results (
                                            id              uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
                                            session_id      uuid REFERENCES sessions(id) ON DELETE CASCADE,
                                            score           integer NOT NULL,
                                            interpretation  text,
                                            recommendations text[],
                                            created_at      timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE test_results ADD CONSTRAINT test_results_session_id_unique UNIQUE (session_id);