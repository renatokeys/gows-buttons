CREATE TABLE gows_chats
(
    -- Unique identifier for the chat (e.g., user or group)
    jid VARCHAR(100) NOT NULL,
    -- Chat Name
    name TEXT,
    -- Last Message timestamp
    conversation_timestamp TIMESTAMP,
    -- Message data
    data TEXT NOT NULL,
    -- Primary key
    PRIMARY KEY (jid)
);

-- Index for jid (useful if filtering by jid)
CREATE INDEX gows_chats_jid_idx ON gows_chats (jid);

-- Index for conversation_timestamp (useful for retrieving chats by time)
CREATE INDEX gows_chats_conversation_timestamp_idx ON gows_chats (conversation_timestamp);
