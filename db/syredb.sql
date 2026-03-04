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

CREATE TYPE user_account_status AS ENUM ('active', 'deactivated');
CREATE TABLE IF NOT EXISTS user_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    account_status user_account_status DEFAULT 'active' NOT NULL,
    email VARCHAR(256) NOT NULL UNIQUE,
    name VARCHAR(512) NOT NULL
);

CREATE TABLE IF NOT EXISTS _db_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL, 
    description TEXT
);
INSERT INTO _db_permission_ VALUES
    ('owner', 'Owner', 'Full permissions'),
    ('add_user', 'Add user', 'Add new users'),
    ('modify_user', 'Modify users', 'Modify users'),
    ('create_data_schema', 'Create data schema', 'Create new data schemas'),
    ('modify_data_schema', 'Modify data schema', 'Modify existing data schema'),
    ('create_project', 'Create project', 'Create new projects'),
    ('create_transform', 'Create transofrm', 'Create new transforms');

CREATE TABLE IF NOT EXISTS db_user_permission_ (
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _db_permission_(_id) NOT NULL,
    PRIMARY KEY (_user, _permission)
);

CREATE OR REPLACE FUNCTION enforce_at_least_one_db_owner()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM db_user_permission_ as p 
        JOIN user_ as u ON p._user=u._id 
        WHERE p._permission = 'owner' AND u.account_status='active'
    ) THEN
        RAISE EXCEPTION 'NO_ACTIVE_USER_WITH_OWNER_ROLE';
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER db_must_have_owner
AFTER UPDATE OR DELETE ON db_user_permission_
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE PROCEDURE enforce_at_least_one_db_owner();

CREATE TABLE IF NOT EXISTS user_auth_ (
    _id UUID REFERENCES user_(_id) NOT NULL UNIQUE,
    auth VARCHAR(2048) NOT NULL
);

CREATE TABLE IF NOT EXISTS _user_session_ (
    _token UUID DEFAULT uuidv7() PRIMARY KEY,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _expires TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    active boolean DEFAULT true NOT NULL
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
    visibility visibility DEFAULT 'private' NOT NULL,
    frozen boolean DEFAULT false NOT NULL -- indicates no more changes are allowed to the sample
);

CREATE TABLE IF NOT EXISTS sample_property_ (
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_sample, _key)
);

CREATE TABLE IF NOT EXISTS sample_property_history_ (
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _version INT NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_sample, _key, _version)
);

CREATE TABLE IF NOT EXISTS sample_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL,
    content TEXT NOT NULL
);


CREATE TABLE IF NOT EXISTS sample_note_history_ (
    _note UUID REFERENCES sample_note_(_id) NOT NULL,
    _version INT NOT NULL,
    _content TEXT NOT NULL,
    _editor UUID REFERENCES user_(_id) NOT NULL,
    _timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
    PRIMARY KEY (_note, _version)
);

CREATE TABLE IF NOT EXISTS _sample_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);
INSERT INTO _sample_permission_ (_id, label, description) VALUES
    ('owner', 'Owner', 'Full permissions'),
    ('read', 'Read', 'Sample is visible'), 
    ('add_data', 'Add data', 'Can add data to the sample'), 
    ('create_note', 'Create note', 'Can create notes for the sample'), 
    ('modify_properties', 'Modify properties', 'Can add, remove, and modify sample properties');

CREATE TABLE IF NOT EXISTS sample_user_permission_ (
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _sample_permission_(_id) NOT NULL,
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

CREATE TABLE IF NOT EXISTS raw_data_type_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS raw_data_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _path VARCHAR(1024) NOT NULL,
    _type UUID REFERENCES raw_data_type_(_id) NOT NULL,
    _filename VARCHAR(256),
    label VARCHAR(128),
    timestamp TIMESTAMP(3) WITH TIME ZONE NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL
);

CREATE TABLE IF NOT EXISTS raw_data_property_ (
    _raw_data UUID REFERENCES raw_data_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_raw_data, _key)
);

CREATE TABLE IF NOT EXISTS raw_data_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _raw_data UUID REFERENCES raw_data_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL, 
    visibility visibility DEFAULT 'private' NOT NULL,
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS _raw_data_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) NOT NULL UNIQUE,
    description TEXT
);
INSERT INTO _raw_data_permission_ (_id, label, description) VALUES
    ('owner', 'Owner', 'Full permission'),
    ('read', 'Read', 'Data is visible'), 
    ('create_note', 'Create note', 'Create notes on the data'), 
    ('modify_properties', 'Modify properties', 'Add, remove, and modify data properties');

CREATE TABLE IF NOT EXISTS raw_data_user_permission_ (
    _raw_data UUID REFERENCES raw_data_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _raw_data_permission_(_id) NOT NULL,
    PRIMARY KEY (_raw_data, _user)
);

CREATE OR REPLACE FUNCTION enforce_raw_data_has_owner()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE raw_data_id UUID;
BEGIN raw_data_id := COALESCE(NEW._raw_data, OLD._raw_data);
    IF NOT EXISTS (
        SELECT 1 FROM raw_data_user_permission_
        WHERE _raw_data = raw_data_id AND _permission = 'owner'
    ) THEN
        RAISE EXCEPTION 'RAW_DATA_WITH_NO_OWNER (%s)', raw_data_id;
    END IF;
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER raw_data_must_have_owner
AFTER INSERT OR UPDATE OR DELETE
ON raw_data_user_permission_
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_raw_data_has_owner();

CREATE TABLE IF NOT EXISTS project_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    label VARCHAR(512) NOT NULL,
    description TEXT,
    visibility visibility DEFAULT 'private' NOT NULL
);

CREATE TABLE IF NOT EXISTS _project_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);
INSERT INTO _project_permission_ (_id, label, description) VALUES 
    ('owner', 'Owner', 'Full permission'),
    ('create_sample', 'Create samples', 'Create samples in the project') 
    ('read', 'Read', 'Project is visible');

CREATE TABLE IF NOT EXISTS project_user_permission_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _project_permission_(_id) NOT NULL,
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

CREATE TABLE IF NOT EXISTS project_sample_note_history_ (
    _note UUID REFERENCES project_sample_note_(_id) NOT NULL,
    _version INT NOT NULL,
    _content TEXT NOT NULL,
    _editor UUID REFERENCES user_(_id) NOT NULL,
    _timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
    PRIMARY KEY (_note, _version)
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


CREATE TABLE IF NOT EXISTS _project_sample_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);
INSERT INTO _project_sample_permission_ (_id, label, description) VALUES
    ('modify_label', 'Modify label', 'Change the sample label'),
    ('modify_tags', 'Modify tags', 'Add or remove tags'),
    ('modify_properties', 'Modify properties', 'Add, remove, or change sample properties');

CREATE TABLE IF NOT EXISTS project_sample_user_permission_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _sample UUID REFERENCES sample_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _project_sample_permission_(_id) NOT NULL,
    PRIMARY KEY (_project, _sample, _user, _permission)
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

CREATE TABLE IF NOT EXISTS transform_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _source UUID REFERENCES data_schema_(_id) NOT NULL,
    _destination UUID REFERENCES data_schema_(_id) NOT NULL,
    _script VARCHAR(1024) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    label VARCHAR(128) NOT NULL,
    description TEXT
);

CREATE TYPE transform_job_status AS ENUM (
    'pending',
    'running',
    'completed',
    'failed'
);

CREATE TABLE IF NOT EXISTS _transform_queue_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _transform UUID REFERENCES transform_(_id) NOT NULL,
    _payload UUID NOT NULL, -- id of the data in the schema table
    status transform_job_status DEFAULT 'pending' NOT NULL,
    started TIMESTAMP WITH TIME ZONE,
    finished TIMESTAMP WITH TIME ZONE,
    error TEXT
);
CREATE INDEX IF NOT EXISTS _transform_queue__status ON _transform_queue_ (status);

CREATE OR REPLACE FUNCTION notify_transform_job()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_notify(
        'new_transform_job',
        json_build_object(
            'id', NEW.id
        )::text
    );

    RETURN NEW;
END;
$$;

CREATE TRIGGER new_transform_job
AFTER INSERT ON _transform_queue_
FOR EACH ROW
EXECUTE FUNCTION notify_transform_job();