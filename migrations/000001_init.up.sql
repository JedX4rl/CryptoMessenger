CREATE TABLE IF NOT EXISTS users
(
    user_id       UUID PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chats
(
    chat_id      UUID PRIMARY KEY,
    name         TEXT        NOT NULL,
    algorithm    TEXT        NOT NULL, -- "RC5", "RC6" и т.д.
    mode         TEXT        NOT NULL, -- "CBC", "OFB", ...
    padding      TEXT        NOT NULL, -- "PKCS7", ...
    iv           TEXT        NOT NULL,
    random_delta TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS room_participants
(
    room_id   UUID REFERENCES chats (chat_id) ON DELETE CASCADE,
    user_id   UUID REFERENCES users (user_id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id)
);