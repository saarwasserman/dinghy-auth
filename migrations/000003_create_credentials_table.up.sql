CREATE TABLE IF NOT EXISTS credentials(
    user_id bigint PRIMARY KEY NOT NULL,
    password_hash bytea NOT NULL
);
