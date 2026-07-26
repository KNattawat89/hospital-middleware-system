-- +goose Up
-- +goose StatementBegin
ALTER TABLE staffs ALTER COLUMN firstname DROP NOT NULL;
ALTER TABLE staffs ALTER COLUMN lastname DROP NOT NULL;
ALTER TABLE staffs ALTER COLUMN email DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE staffs ALTER COLUMN firstname SET NOT NULL;
ALTER TABLE staffs ALTER COLUMN lastname SET NOT NULL;
ALTER TABLE staffs ALTER COLUMN email SET NOT NULL;
-- +goose StatementEnd
