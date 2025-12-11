-- CREATE DATABASE syredb;

CREATE TABLE IF NOT EXISTS user_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    email VARCHAR(256) NOT NULL UNIQUE,
    name VARCHAR(512) NOT NULL,
    permission_roles VARCHAR(256)[] DEFAULT '{"standard_user"}' NOT NULL
);

CREATE TABLE IF NOT EXISTS user_auth_ (
    _id UUID REFERENCES user_(_id) NOT NULL UNIQUE,
    auth VARCHAR(2048) NOT NULL,
    tokens VARCHAR(1024)[]
);

CREATE TABLE IF NOT EXISTS sample_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _owner UUID REFERENCES user_(_id) NOT NULL,
    label VARCHAR(512) NOT NULL
);

CREATE TABLE IF NOT EXISTS sample_group_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    _owner UUID REFERENCES user_(_id) NOT NULL,
    label VARCHAR(512) NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS sample_tag_ (
    sample UUID REFERENCES sample_(_id) NOT NULL,
    tag VARCHAR(512) NOT NULL,
    UNIQUE (sample, tag)
);

CREATE TABLE IF NOT EXISTS sample_note_ (
    _id UUID DEFAULT uuidv7() PRIMARY KEY,
    sample UUID REFERENCES sample_(_id) NOT NULL,
    note TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sample_group_membership_ (
    sample_group UUID REFERENCES sample_group_(_id) NOT NULL,
    sample UUID REFERENCES sample_(_id) NOT NULL,
    UNIQUE (sample_group, sample)
);

CREATE TABLE IF NOT EXISTS sample_property_ (
    sample UUID REFERENCES sample_(_id) NOT NULL,
    key VARCHAR(512) NOT NULL,
    value JSONB NOT NULL,
    PRIMARY KEY (sample, key)
);

CREATE TABLE IF NOT EXISTS sample_group_relation_ (
    parent UUID REFERENCES sample_group_(_id) NOT NULL,
    child UUID REFERENCES sample_group_(_id) NOT NULL,
    UNIQUE (parent, child)
);