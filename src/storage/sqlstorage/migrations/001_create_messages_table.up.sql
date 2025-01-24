-- Create the gows_messages table
CREATE TABLE gows_messages (
                               jid VARCHAR(100) NOT NULL,        -- Unique identifier for the chat (e.g., user or group)
                               id VARCHAR(100) NOT NULL          -- Unique identifier for the message within a chat
                               timestamp TIMESTAMP NOT NULL,     -- Message timestamp
                               from_me BOOLEAN NOT NULL,         -- Whether the message was sent by the user
                               data TEXT NOT NULL,               -- Message data
                               PRIMARY KEY (id)                  -- Primary key
);

-- Index for id (useful if filtering by id)
CREATE INDEX gows_messages_id_idx ON gows_messages (id);

-- Index for jid + id (useful for quickly accessing messages within a chat)
CREATE INDEX gows_messages_jid_id_idx ON gows_messages (jid, id);

-- Index for jid + timestamp (useful for retrieving messages by time within a chat)
CREATE INDEX gows_messages_jid_timestamp_idx ON gows_messages (jid, timestamp);

-- Index for timestamp (useful for retrieving messages across all chats by time)
CREATE INDEX gows_messages_timestamp_idx ON gows_messages (timestamp);
