from typing import Any
import os
import sys
from enum import StrEnum
import uuid
import json
import numpy
import pandas


class Storage(StrEnum):
    Internal = "internal"
    External = "external"


class Args:
    def __init__(self):
        self._token = sys.argv[1]
        self._storage = Storage(sys.argv[2])
        self._data_path = sys.argv[3]

    @property
    def token(self) -> str:
        return self._token

    @property
    def storage(self) -> Storage:
        return self._storage

    @property
    def path(self) -> str:
        return self._data_path


class Data:
    """Data.

    Example:
        ```
        import syredb

        data = syredb.get_data()
        df = data.as_pandas()
        df_avg = df.sum() / data.properties["sample_count"]
        syredb.insert(df_avg)
        ```
    """

    def __init__(
        self,
        token: str,
        storage: Storage,
        data_path: str,
        tags: set,
        properties: dict[str, Any],
    ):
        self.__token = token
        self.__storage = storage
        self.__data_path = data_path
        self._tags = set()
        self._properties = {}

    @property
    def path(self) -> str:
        """Get the path to the data file.

        Returns:
            str: Path to the data file.
        """
        return self.__data_path

    @property
    def properties(self) -> dict[str, Any]:
        """Properties associated with the sample data.
        These include those inherited from the sample and groups.
        """
        return self._properties

    def as_pandas(self) -> pandas.DataFrame:
        if self.__storage != Storage.Internal:
            raise NotImplementedError(
                "data type can not be represented as a dataframe; use `.path` to load data yourself"
            )

        return pandas.read_feather(self.path)


def get_data() -> Data:
    """Get the sample data.

    Returns:
        SampleData: Sample data.
    """
    args = Args()
    with open(args.path) as f:
        data = json.load(f)

    return Data(
        args.token,
        args.storage,
        data["path"],
        data["tags"],
        data["properties"],
    )


# TODO: Should only be callable once.
# Raise exception if called more than once.
# TODO: Accept file input for external storage.
# TODO: Accept other tabular data sources.
def insert(data: pandas.DataFrame):
    """Insert new data into the database.

    Args:
        data (pandas.DataFrame): Data to insert.
    """
    args = Args()
    raise NotImplementedError("TODO")


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
