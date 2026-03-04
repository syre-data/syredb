from typing import Any
import os
import sys
import uuid
import json
import psycopg
import psycopg.sql
import numpy
import pandas


class SampleData:
    """The active sample data.

    Example:
        ```python
        from syredb import transform

        data = transform.get_data()
        df = pd.from_csv(data.path)
        results = data.max()
        transform.insert(results)
        ```
    """

    def __init__(
        self,
        token: str,
        id: uuid.UUID,
        data_path: str,
        tags: set,
        properties: dict[str, Any],
    ):
        self.__token = token
        self.__id = id
        self._data_path = data_path
        self._tags = set()
        self._properties = {}

    @property
    def path(self) -> str:
        """Get the path to the data file.

        Returns:
            str: Path to the data file.
        """
        return self._data_path

    @property
    def properties(self) -> dict[str, Any]:
        """Properties associated with the sample data.
        These include those inherited from the sample and groups.
        """
        return self._properties


def get_data() -> SampleData:
    """Get the sample data.

    Returns:
        SampleData: Sample data.
    """
    token = sys.argv[1]
    sample_data_id = uuid.UUID(sys.argv[2])
    data_path = sys.argv[3]
    with open(data_path) as f:
        sample_data = json.load(f)

    return SampleData(
        token,
        sample_data_id,
        sample_data["data_path"],
        set(),
        sample_data["properties"],
    )


# TODO: Should only be callable once.
# Raise exception if called more than once.
def insert(data: pandas.DataFrame):
    """Insert new data into the database.

    Args:
        data (pandas.DataFrame): The data to insert.
    """
    token = sys.argv[1]
    sample_data_id = uuid.UUID(sys.argv[2])
    output_schema_id = uuid.UUID(sys.argv[4])
    transform_id = uuid.UUID(sys.argv[5])
    with psycopg.connect("dbname=syredb user=postgres password=root") as conn:
        with conn.cursor() as cur:
            record = cur.execute(
                "SELECT _schema, _storage FROM data_schema_ WHERE _id=%s",
                (str(output_schema_id),),
            ).fetchone()
            if record is None:
                exit(1)
            (schema, storage) = record

            try:
                parsed = parse_data_to_schema(data, schema)
            except:
                exit(2)

            # TODO: Account for storage
            table_name = data_storage_table_name_from_id(output_schema_id)
            query = psycopg.sql.SQL(
                "INSERT INTO {table} (_input, _transform, _sample_data, {data_columns}) VALUES ({input}, {transform}, {sample_data}, {data_values})"
            ).format(
                table=psycopg.sql.Identifier(table_name),
                data_columns=psycopg.sql.SQL(", ").join(
                    map(lambda col: psycopg.sql.Identifier(col["label"]), schema)
                ),
                input=psycopg.sql.Literal(str(sample_data_id)),
                transform=psycopg.sql.Literal(str(transform_id)),
                sample_data=psycopg.sql.Literal(str(sample_data_id)),
                data_values=psycopg.sql.SQL(", ").join(
                    psycopg.sql.Placeholder() for _ in schema
                ),
            )

            values = [parsed[col["label"]] for col in schema]
            cur.execute(query, values)
            conn.commit()


def parse_data_to_schema(
    df: pandas.DataFrame, schema: list[dict[str, str]]
) -> dict[str, list]:
    data = {}
    for idx, col in enumerate(schema):
        label = col["label"]
        col_data = df.iloc[idx]
        if col_data is None:
            exit(3)
        data[label] = col_data.to_list()
    return data


def data_storage_table_name_from_id(storage_table: uuid.UUID) -> str:
    id_str = str(storage_table).replace("-", "_")
    return f"data_storage_{id_str}_"
