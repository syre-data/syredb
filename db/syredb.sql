-- CREATE DATABASE syredb;

CREATE TABLE IF NOT EXISTS _app_data_ (
    key VARCHAR(512) PRIMARY KEY,
    value TEXT
);

CREATE TYPE visibility AS ENUM ('private', 'public');
CREATE TYPE property_type AS ENUM (
    'string', 
    'int', 
    'uint', 
    'float', 
    'boolean', 
    'quantity', 
    'timestamp'
);

CREATE TABLE IF NOT EXISTS user_role_ (
    name VARCHAR(64) PRIMARY KEY
);
INSERT INTO user_role_ VALUES
    ('owner'), 
    ('admin'), 
    ('user')
ON CONFLICT (name) DO NOTHING;

CREATE TYPE user_account_status AS ENUM ('active', 'disabled');
CREATE TABLE IF NOT EXISTS user_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    account_status user_account_status DEFAULT 'active' NOT NULL,
    email VARCHAR(256) NOT NULL UNIQUE,
    name VARCHAR(512) NOT NULL,
    role VARCHAR(64) REFERENCES user_role_(name) DEFAULT 'user' NOT NULL
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

CREATE TYPE data_storage AS ENUM ('internal', 'external');
CREATE TYPE data_type AS ENUM ('string', 'int', 'uint', 'float', 'boolean', 'timestamp');

CREATE TABLE IF NOT EXISTS data_schema_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _schema JSONB NOT NULL,
    _storage data_storage NOT NULL,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);

CREATE OR REPLACE FUNCTION data_schema_schema_is_valid(_schema jsonb)
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
CHECK (data_schema_schema_is_valid(_schema));

CREATE TABLE IF NOT EXISTS sample_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL
);

CREATE TABLE IF NOT EXISTS sample_property_ (
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_sample, _key)
);

CREATE TABLE IF NOT EXISTS sample_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL,
    content TEXT NOT NULL
);

CREATE TYPE sample_user_permission AS ENUM (
    'owner',
    'read', 
    'add_data', 
    'create_note', 
    'modify_properties'
);
CREATE TABLE IF NOT EXISTS sample_user_permission_ (
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission sample_user_permission NOT NULL,
    PRIMARY KEY (_sample, _user, _permission)
);

CREATE OR REPLACE FUNCTION enforce_sample_has_owner()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE sample_id UUID;
BEGIN sample_id := COALESCE(NEW._sample, OLD._sample);
    IF NOT EXISTS (
        SELECT 1 FROM sample_user_permission_
        WHERE _sample = sample_id AND _permission = 'owner'
    ) THEN
        RAISE EXCEPTION 'SAMPLE_WITH_NO_OWNER (%s)', sample_id;
    END IF;
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER sample_must_have_owner
AFTER INSERT OR UPDATE OR DELETE
ON sample_user_permission_
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_sample_has_owner();

CREATE TABLE IF NOT EXISTS sample_data_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _schema UUID REFERENCES data_schema_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL
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
    _sample_data UUID REFERENCES sample_data_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL, 
    visibility visibility DEFAULT 'private' NOT NULL,
    content TEXT NOT NULL
);

CREATE TYPE sample_data_user_permission AS ENUM (
    'owner',
    'read', 
    'create_note', 
    'modify_properties'
);
CREATE TABLE IF NOT EXISTS sample_data_user_permission_ (
    _sample_data UUID REFERENCES sample_data_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    permissions sample_data_user_permission[] NOT NULL,
    PRIMARY KEY (_sample_data, _user)
);

CREATE OR REPLACE FUNCTION enforce_sample_data_has_owner()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE sample_data_id UUID;
BEGIN sample_data_id := COALESCE(NEW._sample_data, OLD._sample_data);
    IF NOT EXISTS (
        SELECT 1 FROM sample_data_user_permission_
        WHERE _sample_data = sample_data_id AND _permission = 'owner'
    ) THEN
        RAISE EXCEPTION 'SAMPLE_DATA_WITH_NO_OWNER (%s)', sample_data_id;
            -- USING ERRCODE = '23514'; -- check_violation
    END IF;
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER sample_data_must_have_owner
AFTER INSERT OR UPDATE OR DELETE
ON sample_data_user_permission_
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_sample_data_has_owner();

CREATE TABLE IF NOT EXISTS project_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    label VARCHAR(512) NOT NULL,
    description TEXT,
    visibility visibility DEFAULT 'private' NOT NULL
);

CREATE TYPE project_user_permission AS ENUM (
    'owner',
    'admin', 
    'read_write', 
    'read'
);
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

CREATE TABLE IF NOT EXISTS project_sample_tag_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _tag VARCHAR(64) NOT NULL,
    PRIMARY KEY (_project, _sample, _tag)
);

CREATE TABLE IF NOT EXISTS project_sample_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _project UUID REFERENCES project_(_id) NOT NULL,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL,
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