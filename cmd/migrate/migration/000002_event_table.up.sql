CREATE TABLE events (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    location TEXT,
    user_id UUID REFERENCES users(id),
    date_time TIMESTAMP NOT NULL
);