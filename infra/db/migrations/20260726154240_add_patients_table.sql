-- +goose Up
-- +goose StatementBegin
CREATE TABLE patients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hospital_id UUID NOT NULL REFERENCES hospitals(id),
    national_id VARCHAR(13) UNIQUE,
    passport_id VARCHAR(9) UNIQUE,
    patient_hn VARCHAR(20) UNIQUE,
    firstname_th VARCHAR(100),
    middlename_th VARCHAR(100),
    lastname_th VARCHAR(100),
    firstname_en VARCHAR(100),
    middlename_en VARCHAR(100),
    lastname_en VARCHAR(100),
    date_of_birth DATE,
    phone_number VARCHAR(10),
    email VARCHAR(100) UNIQUE,
    gender VARCHAR(1) CHECK (gender IN ('M', 'F')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX patients_hospital_id_national_id_idx ON patients (hospital_id, national_id);
CREATE UNIQUE INDEX patients_hospital_id_passport_id_idx ON patients (hospital_id, passport_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE patients;
-- +goose StatementEnd
