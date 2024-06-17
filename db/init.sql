CREATE TABLE IF NOT EXISTS urls (
     id BINARY(7) NOT NULL,
     long_url VARCHAR(255) UNIQUE NOT NULL,
     PRIMARY KEY (id),
     INDEX (long_url)
) ENGINE = RocksDB DEFAULT COLLATE = ascii_bin;
