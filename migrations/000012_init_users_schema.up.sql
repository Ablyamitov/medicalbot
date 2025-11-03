CREATE TABLE IF NOT EXISTS users (
                       id          UUID DEFAULT gen_random_uuid() PRIMARY KEY,
                       chat_id     VARCHAR(50) UNIQUE NOT NULL,  -- Telegram chat_id (stringified)
                       username    VARCHAR(255),
                       first_name  VARCHAR(255),
                       last_name   VARCHAR(255),
                       created_at  TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL
);
