-- Test SQL with JOINs
CREATE TABLE users (id INTEGER, name TEXT, age INTEGER);
CREATE TABLE orders (id INTEGER, user_id INTEGER, product TEXT, amount INTEGER);

INSERT INTO users (id, name, age) VALUES (1, 'Alice', 30);
INSERT INTO users (id, name, age) VALUES (2, 'Bob', 25);
INSERT INTO users (id, name, age) VALUES (3, 'Charlie', 35);

INSERT INTO orders (id, user_id, product, amount) VALUES (1, 1, 'Laptop', 999);
INSERT INTO orders (id, user_id, product, amount) VALUES (2, 1, 'Mouse', 29);
INSERT INTO orders (id, user_id, product, amount) VALUES (3, 2, 'Keyboard', 79);
INSERT INTO orders (id, user_id, product, amount) VALUES (4, 3, 'Monitor', 299);

-- Simple SELECT
SELECT * FROM users;

-- JOIN query
SELECT users.name, orders.product, orders.amount
FROM users
JOIN orders ON users.id = orders.user_id;

-- JOIN with WHERE
SELECT users.name, orders.product
FROM users
JOIN orders ON users.id = orders.user_id
WHERE orders.amount > 50;
