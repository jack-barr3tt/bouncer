ALTER TABLE user_apps ADD COLUMN can_share boolean NOT NULL DEFAULT false;

ALTER TABLE users DROP COLUMN can_share;
