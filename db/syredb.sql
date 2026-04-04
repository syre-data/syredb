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
    email VARCHAR(256) NOT NULL UNIQUE,
    name VARCHAR(512) NOT NULL,
    account_status user_account_status DEFAULT 'active' NOT NULL
);

CREATE TABLE IF NOT EXISTS _db_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL, 
    description TEXT
);
INSERT INTO _db_permission_ VALUES
    ('owner', 'Owner', 'Full permissions'),
    ('user_create', 'Create users', 'Create new users'),
    ('user_modify', 'Modify users', 'Modify users'),
    ('data_schema_create', 'Create data schema', 'Create new data schemas'),
    ('data_schema_modify', 'Modify data schema', 'Modify existing data schema'),
    ('data_type_create', 'Create data types', 'Create new data types'),
    ('data_type_modify', 'Modify data types', 'Modify existing data types'),
    ('ingestion_script_create', 'Create ingestion script', 'Create ingestion scripts'),
    ('ingestion_script_modify', 'Modify ingestion script', 'Modify ingestion scripts'),
    ('data_type_transform_create', 'Create data type transform', 'Create new data type transforms'),
    ('data_type_transform_modify', 'Modify data type transforms', 'Modify existing data type transforms'),
    ('project_create', 'Create project', 'Create new projects');

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

CREATE TYPE value_type AS ENUM (
    'string', 
    'int', 
    'uint', 
    'float', 
    'boolean', 
    'timestamp'
);

CREATE TYPE data_schema_cardinality AS ENUM (
    'single',
    'multiple'
);

CREATE TABLE IF NOT EXISTS data_schema_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _cardinality data_schema_cardinality NOT NULL,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS data_schema_field_ (
    _id UUID REFERENCES data_schema_(_id) NOT NULL,
    _label VARCHAR(128) NOT NULL,
    _dtype value_type NOT NULL, 
    description TEXT,
    PRIMARY KEY (_id, _label)
);

CREATE TYPE data_source_storage AS ENUM (
    'internal', 
    'external'
);

CREATE TABLE IF NOT EXISTS data_type_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _storage data_source_storage NOT NULL,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT,
    active boolean DEFAULT true NOT NULL
);

CREATE TABLE IF NOT EXISTS data_type_internal_storage_ (
    _data_type UUID REFERENCES data_type_(_id) PRIMARY KEY,
    _schema UUID REFERENCES data_schema_(_id)
);

CREATE TYPE data_type_external_source_cardinality AS ENUM (
    'single',
    'multiple'
);

CREATE TABLE IF NOT EXISTS data_type_external_source_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _data_type UUID REFERENCES data_type_(_id) NOT NULL,
    _label VARCHAR(128) NOT NULL,
    _required boolean NOT NULL,
    _cardinality data_type_external_source_cardinality NOT NULL,
    description TEXT,
    ext_filter VARCHAR(64)[],
    UNIQUE (_data_type, _label)
);

CREATE TABLE IF NOT EXISTS ingestion_script_cmd_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _path VARCHAR(2048) NOT NULL,
    _cmd VARCHAR(1024),
    _args VARCHAR(64)[] DEFAULT array[]::varchar[]
);

CREATE TABLE IF NOT EXISTS ingestion_script_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _type UUID REFERENCES data_type_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    cmd UUID REFERENCES ingestion_script_cmd_(_id) NOT NULL,
    label VARCHAR(128) NOT NULL UNIQUE,
    description TEXT
);

CREATE TYPE data_creator_type AS ENUM (
    'user',
    'transform'
);

CREATE TABLE IF NOT EXISTS data_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _type UUID REFERENCES data_type_(_id) NOT NULL,
    _creator_type data_creator_type NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL
);

CREATE TABLE IF NOT EXISTS data_property_ (
    _data UUID REFERENCES data_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_data, _key)
);

CREATE TABLE IF NOT EXISTS data_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _data UUID REFERENCES data_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL, 
    visibility visibility DEFAULT 'private' NOT NULL,
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS _data_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) NOT NULL UNIQUE,
    description TEXT
);
INSERT INTO _data_permission_ (_id, label, description) VALUES
    ('owner', 'Owner', 'Full permission'),
    ('read', 'Read', 'Data is visible'), 
    ('note_create', 'Create note', 'Create notes on this data'),
    ('properties_modify', 'Modify properties', 'Add, remove, and modify data properties');

CREATE TABLE IF NOT EXISTS data_user_permission_ (
    _data UUID REFERENCES data_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _data_permission_(_id) NOT NULL,
    PRIMARY KEY (_data, _user)
);

CREATE OR REPLACE FUNCTION enforce_data_has_owner()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE data_id UUID;
BEGIN data_id := COALESCE(NEW._data, OLD._data);
    IF NOT EXISTS (
        SELECT 1 FROM data_user_permission_
        WHERE _data = data_id AND _permission = 'owner'
    ) THEN
        RAISE EXCEPTION 'DATA_WITH_NO_OWNER (%s)', data_id;
    END IF;
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER data_must_have_owner
AFTER INSERT OR UPDATE OR DELETE
ON data_user_permission_
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_data_has_owner();

CREATE TYPE data_type_transform_job_status AS ENUM (
    'pending',
    'running',
    'completed',
    'failed'
);

CREATE TABLE IF NOT EXISTS data_source_external_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _data UUID REFERENCES data_(_id) NOT NULL,
    _source UUID REFERENCES data_type_external_source_(_id) NOT NULL,
    _path VARCHAR(2048) NOT NULL UNIQUE,
    label VARCHAR(256)
);

CREATE TABLE IF NOT EXISTS data_type_transform_cmd_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _path VARCHAR(2048) NOT NULL UNIQUE,
    _cmd VARCHAR(1024) NOT NULL,
    _args VARCHAR(64)[] DEFAULT array[]::varchar[]
);

CREATE TABLE IF NOT EXISTS data_type_transform_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _type UUID REFERENCES data_type_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _source UUID REFERENCES data_type_(_id) NOT NULL,
    _destination UUID REFERENCES data_type_(_id) NOT NULL,
    cmd UUID REFERENCES data_type_transform_cmd_(_id) NOT NULL,
    label VARCHAR(128) NOT NULL,
    description TEXT
);

-- CREATE TABLE IF NOT EXISTS _transform_script_history_ (
--     _id UUID PRIMARY KEY,
--     _data_type_transform UUID REFERENCES data_type_transform_(_id),
--     _path VARCHAR(1024) NOT NULL UNIQUE,
--     _cmd VARCHAR(1024) NOT NULL,
--     _args VARCHAR(64)[] DEFAULT array[]::varchar[],
--     _expiration TIMESTAMP(3) WITH TIME ZONE NOT NULL
-- );

CREATE TABLE IF NOT EXISTS _data_type_transform_queue_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _transform UUID REFERENCES data_type_transform_(_id) NOT NULL,
    _payload UUID REFERENCES data_(_id) NOT NULL,
    status data_type_transform_job_status DEFAULT 'pending' NOT NULL,
    started TIMESTAMP WITH TIME ZONE,
    finished TIMESTAMP WITH TIME ZONE,
    error TEXT
);
CREATE INDEX IF NOT EXISTS _data_type_transform_queue__status ON _data_type_transform_queue_ (status);

CREATE OR REPLACE FUNCTION notify_data_type_transform_job()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_notify(
        'new_data_type_transform_job',
        json_build_object(
            'id', NEW.id
        )::text
    );

    RETURN NEW;
END;
$$;

CREATE TRIGGER new_data_type_transform_job
AFTER INSERT ON _data_type_transform_queue_
FOR EACH ROW
EXECUTE FUNCTION notify_data_type_transform_job();

CREATE TABLE IF NOT EXISTS data_creator_user_ (
    _data UUID REFERENCES data_(_id) PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL
);

CREATE TABLE IF NOT EXISTS data_creator_transform_ (
    _data UUID REFERENCES data_(_id) PRIMARY KEY,
    _creator UUID REFERENCES data_type_transform_(_id) NOT NULL
);

CREATE TABLE IF NOT EXISTS data_group_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL,
    frozen boolean DEFAULT false NOT NULL -- indicates no more changes are allowed to the group
);

CREATE TABLE IF NOT EXISTS data_group_property_ (
    _group UUID REFERENCES data_group_(_id) NOT NULL,
    _key VARCHAR(512) NOT NULL,
    _type property_type NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (_group, _key)
);

-- CREATE TABLE IF NOT EXISTS _data_group_property_history_ (
--     _group UUID REFERENCES data_group_(_id) NOT NULL,
--     _key VARCHAR(512) NOT NULL,
--     _version INT NOT NULL,
--     value JSONB NOT NULL,
--     PRIMARY KEY (_group, _key, _version)
-- );

CREATE TABLE IF NOT EXISTS data_group_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _group UUID REFERENCES data_group_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL,
    content TEXT NOT NULL
);

-- CREATE TABLE IF NOT EXISTS _data_group_note_history_ (
--     _note UUID REFERENCES data_group_note_(_id) NOT NULL,
--     _version INT NOT NULL,
--     _content TEXT NOT NULL,
--     _editor UUID REFERENCES user_(_id) NOT NULL,
--     _timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
--     PRIMARY KEY (_note, _version)
-- );

CREATE TABLE IF NOT EXISTS _data_group_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);
INSERT INTO _data_group_permission_ (_id, label, description) VALUES
    ('owner', 'Owner', 'Full permissions'),
    ('read', 'Read', 'Data group is visible'), 
    ('membership_modify', 'Modify membership', 'Add or remove data to the group'), 
    ('note_create', 'Create note', 'Create notes for the group'), 
    ('properties_modify', 'Modify properties', 'Add, remove, and modify group properties');

CREATE TABLE IF NOT EXISTS data_group_user_permission_ (
    _group UUID REFERENCES data_group_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _data_group_permission_(_id) NOT NULL,
    PRIMARY KEY (_group, _user, _permission)
);

CREATE OR REPLACE FUNCTION enforce_data_group_has_owner()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE group_id UUID;
BEGIN group_id := COALESCE(NEW._group, OLD._group);
    IF NOT EXISTS (
        SELECT 1 FROM data_group_user_permission_
        WHERE _group = group_id AND _permission = 'owner'
    ) THEN
        RAISE EXCEPTION 'DATA_GROUP_WITH_NO_OWNER (%s)', group_id;
    END IF;
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER data_group_must_have_owner
AFTER INSERT OR UPDATE OR DELETE
ON data_group_user_permission_
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_data_group_has_owner();

-- TODO: Check data does not exist in child of group.
-- If adding data to a child group, move it instead.
CREATE TABLE IF NOT EXISTS data_group_membership_ (
    _group UUID REFERENCES data_group_(_id),
    _data UUID REFERENCES data_(_id),
    PRIMARY KEY(_group, _data)
);

-- TODO: Ensure DAG
CREATE TABLE IF NOT EXISTS data_group_relation_ (
    _parent UUID REFERENCES data_group_(_id) NOT NULL,
    _child UUID REFERENCES data_group_(_id) NOT NULL,
    PRIMARY KEY (_parent, _child)
);

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
    ('data_group_create', 'Create data groups', 'Create data groups in the project'),
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

CREATE TABLE IF NOT EXISTS project_data_group_tag_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _group UUID REFERENCES data_group_(_id) NOT NULL,
    _tag VARCHAR(64) NOT NULL,
    PRIMARY KEY (_project, _group, _tag)
);

CREATE TABLE IF NOT EXISTS project_data_group_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _project UUID REFERENCES project_(_id) NOT NULL,
    _group UUID REFERENCES data_group_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
    visibility visibility DEFAULT 'private' NOT NULL,
    content TEXT NOT NULL
);

-- CREATE TABLE IF NOT EXISTS _project_data_group_note_history_ (
--     _note UUID REFERENCES project_data_group_note_(_id) NOT NULL,
--     _version INT NOT NULL,
--     _content TEXT NOT NULL,
--     _editor UUID REFERENCES user_(_id) NOT NULL,
--     _timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,
--     PRIMARY KEY (_note, _version)
-- );

CREATE TABLE IF NOT EXISTS project_data_group_membership_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _group UUID REFERENCES data_group_(_id) NOT NULL,
    _creator UUID REFERENCES user_(_id) NOT NULL,
    _timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NOW() NOT NULL,  
    label VARCHAR(512) NOT NULL,
    PRIMARY KEY (_project, _group),
    UNIQUE (_project, label)
);

CREATE TABLE IF NOT EXISTS _project_data_group_permission_ (
    _id VARCHAR(128) PRIMARY KEY,
    label VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);
INSERT INTO _project_data_group_permission_ (_id, label, description) VALUES
    ('label_modify', 'Modify label', 'Change the data group label'),
    ('tags_modify', 'Modify tags', 'Add or remove tags'),
    ('properties_modify', 'Modify properties', 'Add, remove, or change group properties');

CREATE TABLE IF NOT EXISTS project_data_group_user_permission_ (
    _project UUID REFERENCES project_(_id) NOT NULL,
    _group UUID REFERENCES data_group_(_id) NOT NULL,
    _user UUID REFERENCES user_(_id) NOT NULL,
    _permission VARCHAR(128) REFERENCES _project_data_group_permission_(_id) NOT NULL,
    PRIMARY KEY (_project, _group, _user, _permission)
);