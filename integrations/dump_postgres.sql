/*Generated by xorm 2020-03-27 13:40:15, from sqlite3 to sqlite3*/

CREATE TABLE IF NOT EXISTS "test_dump_struct" ("id" SERIAL PRIMARY KEY  NOT NULL, "name" TEXT NULL);
INSERT INTO "test_dump_struct" ("id", "name") VALUES (1, '1');
INSERT INTO "test_dump_struct" ("id", "name") VALUES (2, '2
');
INSERT INTO "test_dump_struct" ("id", "name") VALUES (3, '3;');
INSERT INTO "test_dump_struct" ("id", "name") VALUES (4, '4
;
''''');
INSERT INTO "test_dump_struct" ("id", "name") VALUES (5, '5''
');
