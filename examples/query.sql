-- Example SQL script for testing Ditto
-- Creates tables and runs queries

-- Create a users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    age INTEGER
);

-- Insert some data
INSERT INTO users (id, name, email, age) VALUES (1, 'Alice', 'alice@example.com', 30);
INSERT INTO users (id, name, email, age) VALUES (2, 'Bob', 'bob@example.com', 25);
INSERT INTO users (id, name, email, age) VALUES (3, 'Charlie', 'charlie@example.com', 35);

-- Query all users
SELECT * FROM users;

-- Query users over 25
SELECT name, age FROM users WHERE age > 25;

-- Create posts table
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    title TEXT,
    content TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Insert posts
INSERT INTO posts (id, user_id, title, content) VALUES (1, 1, 'Hello World', 'My first post');
INSERT INTO posts (id, user_id, title, content) VALUES (2, 1, 'Ditto is awesome', 'Single binary magic');
INSERT INTO posts (id, user_id, title, content) VALUES (3, 2, 'SQL Power', 'Embedded SQLite!');

-- Join query
SELECT users.name, posts.title 
FROM posts 
JOIN users ON posts.user_id = users.id;
