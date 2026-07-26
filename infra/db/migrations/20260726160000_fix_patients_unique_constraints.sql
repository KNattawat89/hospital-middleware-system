-- +goose Up
-- +goose StatementBegin
ALTER TABLE patients DROP CONSTRAINT patients_national_id_key;
ALTER TABLE patients DROP CONSTRAINT patients_passport_id_key;
ALTER TABLE patients DROP CONSTRAINT patients_patient_hn_key;
ALTER TABLE patients DROP CONSTRAINT patients_email_key;

CREATE UNIQUE INDEX patients_hospital_id_patient_hn_idx ON patients (hospital_id, patient_hn);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX patients_hospital_id_patient_hn_idx;

ALTER TABLE patients ADD CONSTRAINT patients_national_id_key UNIQUE (national_id);
ALTER TABLE patients ADD CONSTRAINT patients_passport_id_key UNIQUE (passport_id);
ALTER TABLE patients ADD CONSTRAINT patients_patient_hn_key UNIQUE (patient_hn);
ALTER TABLE patients ADD CONSTRAINT patients_email_key UNIQUE (email);
-- +goose StatementEnd
