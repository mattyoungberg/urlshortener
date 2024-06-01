CREATE TABLE IF NOT EXISTS urls (
     id BINARY(7) NOT NULL,
     short_url CHAR(10) NOT NULL,
     long_url VARCHAR(255) NOT NULL,
     PRIMARY KEY (id),
     INDEX (short_url),
     INDEX (long_url)
);
