/* see https://github.com/pingcap/tiflow/issues/11704 */
CREATE TABLE GENERATED_TABLE (
  id int PRIMARY KEY,
  A SMALLINT UNSIGNED,
  B SMALLINT UNSIGNED AS (2 * A) STORED,
  C SMALLINT UNSIGNED AS (3 * A) STORED NOT NULL
);
INSERT INTO GENERATED_TABLE VALUES (1, 15, DEFAULT, DEFAULT);