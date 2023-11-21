-- Create a database
SELECT 'CREATE DATABASE books'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'books')\gexec

\c books;
    
--You need to check if the table exists
SELECT 'CREATE TABLE books (
        id serial PRIMARY KEY,
        title VARCHAR (50) UNIQUE NOT NULL,
        author VARCHAR (100) UNIQUE NOT NULL
    )'
WHERE NOT EXISTS (SELECT * FROM pg_catalog.pg_tables WHERE tablename = 'books')\gexec    

INSERT INTO books (title, author) VALUES
    ('Book 3', 'Author 3'),
    ('Book 4', 'Author 4');

SELECT * FROM books;

SELECT 'CREATE TABLE users (
        id serial PRIMARY KEY,
        username VARCHAR (50) UNIQUE NOT NULL,
        password VARCHAR (100) NOT NULL
    )'
WHERE NOT EXISTS (SELECT * FROM pg_catalog.pg_tables WHERE tablename = 'users')\gexec    

INSERT INTO users ("username", "password") VALUES
    ('test', 'test');

SELECT * FROM users WHERE "username" = 'test';