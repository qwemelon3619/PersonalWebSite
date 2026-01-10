-- Dummy posts
INSERT INTO posts (title, content, author_id, author_name, published, published_at, created_at, updated_at)
VALUES
  ('First Post', 'Hello, this is Alice''s first post!', 1, 'alice', true, NOW(), NOW(), NOW()),
  ('Bob''s Thoughts', 'Bob shares his thoughts on Go.', 2, 'bob', true, NOW(), NOW(), NOW()),
  ('Unpublished Draft', 'This is a draft by Charlie.', 3, 'charlie', false, NULL, NOW(), NOW());
