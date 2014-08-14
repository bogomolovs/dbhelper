CREATE DATABASE test
  WITH OWNER = test
       ENCODING = 'UTF8'
       TABLESPACE = pg_default
       LC_COLLATE = 'English_United Kingdom.1252'
       LC_CTYPE = 'English_United Kingdom.1252'
       CONNECTION LIMIT = -1;

CREATE TABLE test
(
  id bigserial NOT NULL,
  text character varying(255) NOT NULL,
  b boolean NOT NULL,
  c bigint NOT NULL,
  m bigint NOT NULL,
  CONSTRAINT test_pkey PRIMARY KEY (id)
)
WITH (
  OIDS=FALSE
);
ALTER TABLE test
  OWNER TO test;
