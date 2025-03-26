-- +goose Up
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT, -- Short summary for the blog list page
    author VARCHAR(255),
    published_at TIMESTAMP WITH TIME ZONE,
    thumbnail VARCHAR(255),
    status VARCHAR(50) DEFAULT 'draft', -- e.g., 'draft', 'published'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    tags VARCHAR(255)[] -- Array of tags (e.g., ['market-update', 'property-tips'])
);

CREATE INDEX idx_posts_slug ON posts (slug);