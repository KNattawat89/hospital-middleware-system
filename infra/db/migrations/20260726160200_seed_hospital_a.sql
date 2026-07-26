-- +goose Up
-- +goose StatementBegin
INSERT INTO hospitals (code, name, api_base_url)
VALUES ('HOSPITAL_A', 'Hospital A', 'https://hospital-a.api.co.th');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM hospitals WHERE code = 'HOSPITAL_A';
-- +goose StatementEnd
