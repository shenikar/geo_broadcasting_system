-- +migrate Down
DROP TABLE IF EXISTS location_checks;
DROP TABLE IF EXISTS incidents;
DROP EXTENSION IF EXISTS "postgis";
DROP EXTENSION IF EXISTS "uuid-ossp";