ALTER TABLE sessions
ADD COLUMN IF NOT EXISTS is_consultation_needed boolean not null default false;