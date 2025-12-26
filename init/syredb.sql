-- CREATE DATABASE syredb;

CREATE TABLE IF NOT EXISTS _app_data_ (
    key VARCHAR(512) PRIMARY KEY,
    value TEXT
);

CREATE TYPE visibility AS ENUM ('private', 'public');
CREATE TYPE property_type AS ENUM ('string', 'int', 'uint', 'float', 'boolean', 'quantity', 'timestamp');

CREATE TYPE user_account_status AS ENUM ('active', 'disabled');
CREATE TYPE user_role AS ENUM ('owner', 'admin', 'user');
CREATE TABLE IF NOT EXISTS user_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    account_status user_account_status DEFAULT 'active' NOT NULL,
    email VARCHAR(256) NOT NULL UNIQUE,
    name VARCHAR(512) NOT NULL,
    role user_role DEFAULT 'user' NOT NULL
);

CREATE OR REPLACE FUNCTION enforce_at_least_one_user_owner()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM user_ WHERE role = 'owner'
    ) THEN
        RAISE EXCEPTION 'NO_USER_WITH_OWNER_ROLE';
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER users_must_have_owner
AFTER UPDATE OR DELETE ON user_
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE PROCEDURE enforce_at_least_one_user_owner();

CREATE TABLE IF NOT EXISTS user_auth_ (
    _id UUID REFERENCES user_(_id) NOT NULL UNIQUE,
    auth VARCHAR(2048) NOT NULL,
    tokens VARCHAR(256)[]
);

CREATE TABLE IF NOT EXISTS project_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    label VARCHAR(512) NOT NULL,
    description TEXT,
    visibility visibility DEFAULT 'private' NOT NULL
);

CREATE TYPE project_user_permission AS ENUM ('read', 'read_write', 'admin', 'owner');
CREATE TABLE IF NOT EXISTS project_user_permission_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    permission project_user_permission NOT NULL,
    PRIMARY KEY (_project, _user)
);

CREATE TABLE IF NOT EXISTS project_tag_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _tag VARCHAR(512) NOT NULL,
    PRIMARY KEY (_project, _tag)
);

CREATE TABLE IF NOT EXISTS project_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _project UUID REFERENCES project_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    note TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sample_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL
);

CREATE TABLE IF NOT EXISTS project_sample_tag_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _tag VARCHAR(64) NOT NULL,
    PRIMARY KEY (_project, _sample, _tag)
);

CREATE TABLE IF NOT EXISTS sample_property_ (
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_sample, _key)
);

CREATE TABLE IF NOT EXISTS project_sample_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _project UUID REFERENCES project_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOT() NOT NULL, 
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS project_sample_membership_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,  
    label VARCHAR(512) NOT NULL,
    PRIMARY KEY (_project, _sample),
    UNIQUE (_project, label)
);

CREATE TABLE IF NOT EXISTS sample_group_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _project UUID REFERENCES project_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    label VARCHAR(512) NOT NULL,
    description TEXT,
    UNIQUE (_project, label)
);

CREATE TABLE IF NOT EXISTS sample_group_relation_ (
    _parent UUID REFERENCES sample_group_(_id) NOT NULL,
    _child UUID REFERENCES sample_group_(_id) NOT NULL,
    PRIMARY KEY (_parent, _child)
);

CREATE TABLE IF NOT EXISTS sample_group_property_ (
    _sample_group UUID REFERENCES sample_group_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    sticky boolean DEFAULT FALSE NOT NULL,
    PRIMARY KEY (_sample_group, _key)
);

CREATE TABLE IF NOT EXISTS sample_group_sample_membership_ (
    _sample_group UUID REFERENCES sample_group_(_id) NOT NULL,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    PRIMARY KEY (_sample_group, _sample)
);

CREATE TYPE data_storage AS ENUM ('internal', 'file');
CREATE TYPE data_type AS ENUM ('string', 'int', 'uint', 'float', 'boolean', 'timestamp');

CREATE TABLE IF NOT EXISTS data_schema_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _schema JSONB NOT NULL,
    _storage data_storage NOT NULL,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);

CREATE OR REPLACE FUNCTION data_schmea_schema_is_valid(_schema jsonb)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT
        jsonb_typeof(_schema) = 'array'
        AND NOT EXISTS (
            SELECT 1
            FROM jsonb_array_elements(_schema) AS elem
            WHERE
                jsonb_typeof(elem) <> 'object'
                OR NOT (elem ? 'label')
                OR NOT (elem ? 'dtype')
                OR jsonb_typeof(elem->'label') <> 'string'
                OR jsonb_typeof(elem->'dtype') <> 'string'
        );
$$;

ALTER TABLE data_schema_
ADD CONSTRAINT schema_is_valid
CHECK (data_schmea_schema_is_valid(_schema));

CREATE TABLE IF NOT EXISTS sample_data_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _schema UUID REFERENCES data_schema_(_id) NOT NULL,
    _project UUID REFERENCES project_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS sample_data_property_ (
    _sample_data UUID REFERENCES sample_data_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_sample_data, _key)
);

CREATE TABLE IF NOT EXISTS sample_data_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _data UUID REFERENCES sample_data_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL, 
    content TEXT NOT NULL
);