-- +goose Up
CREATE TABLE csft (
    token TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY(user_id) REFERENCES users(id)
);