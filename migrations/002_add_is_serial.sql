-- migrations/002_add_is_serial.sql

ALTER TABLE series ADD COLUMN IF NOT EXISTS is_serial BOOLEAN DEFAULT false;
