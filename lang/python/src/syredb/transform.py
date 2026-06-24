# Data type transform functionality.
from typing import Any, Optional, Iterable
import os
import sys
import struct
import socket
from enum import StrEnum
import uuid
import json
import datetime as dt
import numpy
import pandas
from ._socket import _query, LOCALHOST
from ._data import (
    Storage,
    ValueType,
    DataSchemaCardinality,
    DataSchemaField,
    DataSchema,
    Data,
)

__SYREDB_CONNECTION__ = None


class Args:
    def __init__(self):
        self._token = sys.argv[1]
        self._port = int(sys.argv[2])

    @property
    def token(self) -> str:
        return self._token

    @property
    def port(self) -> int:
        return self._port


def _connection() -> socket.socket:
    global __SYREDB_CONNECTION__

    if __SYREDB_CONNECTION__ is None:
        args = Args()
        __SYREDB_CONNECTION__ = socket.create_connection((LOCALHOST, args.port))
    return __SYREDB_CONNECTION__


class InputData:
    """Input data.

    Example:
        ```py
        from syredb.transform import get_data, OutputData

        input = get_data()
        df = input.as_pandas() # only valid for internally stored data
        df_avg = df.sum() / input.properties["sample_count"]

        output = OutputData()
        output.set_data(df_avg) # only valid for internally stored data
        output.save()
        ```

        ```py
        from syredb.transform import get_data, OutputData

        input = get_data()
        img_path = input.data_sources["image"] # only valid for externally stored data
        img = load_image(img_path)
        img_blur = blur(img)

        output = OutputData()
        output.set_source("blurred_image", img_blur) # only valid for externally stored data
        output.save()
        ```
    """

    def __init__(
        self,
        token: str,
        tags: set[str],
        properties: dict[str, Any],
        data_source: Optional[dict[str, str | list[str]]],
        conn: Optional[socket.socket],
    ):
        """Create a new input data.

        Args:
            token (str): Job token.
            storage (Storage): Storage type.
            tags (set): _description_
            properties (dict[str, Any]): _description_
            data_source (Optional[dict[str, str | list[str]]]):
                Data source paths if storage is external, None if internal.
            port (Optional[socket.socket]): Query connection if storage is internal, None if external.
        """
        if data_source is None and conn is None:
            raise ValueError("one of `data_source` or `conn` must be provided")
        if data_source is not None and conn is not None:
            raise ValueError("only one of `data_source` or `conn` can be provided")

        self.__token = token
        self.__data_source = data_source
        self.__conn = conn
        self._tags = tags
        self._properties = properties

    @property
    def properties(self) -> dict[str, Any]:
        """Properties associated with the sample data.
        These include those inherited from the sample and groups.
        """
        return self._properties

    @property
    def tags(self) -> set[str]:
        return self.tags

    @property
    def data_sources(self) -> dict[str, str | list[str]]:
        """Get the paths of the data sources.

        Raises:
            KeyError: Data storage is not external.

        Returns:
            dict[str, str | list[str]: Paths to source file(s) keyed by label.
        """
        if self.__data_source is None:
            raise KeyError("data sources are only available for externally stored data")

        return self.__data_source

    def as_csv(self) -> os.PathLike:
        """Creates a CSV file containing the data's values.

        Raises:
            NotImplementedError: Data is not stored internally.

        Returns:
            os.PathLike: Path to the CSV data file.
        """
        if self.__conn is None:
            raise NotImplementedError(
                "data is not stored internally; use `.data_sources` to load external data"
            )

        cmd = {"token": self.__token, "fn": "values_as_csv"}
        data_path = _query(cmd, self.__conn)
        return data_path

    def as_feather(self) -> os.PathLike:
        """Creates a data file containing the data's values using Apache's Feather format.

        Raises:
            NotImplementedError: Data is not stored internally.

        Returns:
            os.PathLike: Path to the data file.
        """
        if self.__conn is None:
            raise NotImplementedError(
                "data is not stored internally; use `.data_sources` to load external data"
            )

        cmd = {"token": self.__token, "fn": "values_as_feather"}
        data_path = _query(cmd, self.__conn)
        return data_path

    def as_dict(self) -> dict[str, Any]:
        """Get data as a dictionary of label to values.
        If the data's cardinality is `single`, dictionary values are the value.
        If `multiple` values are a list of values.

        Raises:
            NotImplementedError: Data is not stored internally.

        Returns:
            dict[str, Any]: Data values.
        """
        if self.__conn is None:
            raise NotImplementedError(
                "data is not stored internally; use `.data_sources` to load external data"
            )

        cmd = {"token": self.__token, "fn": "values_as_map"}
        data_path = _query(cmd, self.__conn)
        with open(data_path) as f:
            data = json.load(f)
            return data

    def as_pandas(self) -> pandas.DataFrame:
        """Get data as a pandas DataFrame.

        Returns:
            pandas.DataFrame: Data values.
        """
        data_path = self.as_feather()
        return pandas.read_feather(data_path)


class OutputData(Data):
    """Output data.

    Example:
        ```py
        from syredb.transform import get_data, OutputData

        input = get_data()
        df = input.as_pandas() # only valid for internally stored data
        df_avg = df.sum() / input.properties["sample_count"]

        output = OutputData()
        output.add_tag("summary")
        output.set_property("count", 10)
        output.set_values(df_avg) # only valid for internally stored data
        output.save()
        ```

        ```py
        from syredb.transform import get_data, OutputData

        input = get_data()
        img_path = input.data_sources["image"] # only valid for externally stored data
        img = load_image(img_path)
        img_blur = blur(img, radius=10)

        output = OutputData()
        output.add_tag("blur")
        output.set_property("blur_radius", 10)
        output.set_source("blurred_image", img_blur) # only valid for externally stored data
        output.save()
        ```
    """

    def __init__(self):
        args = Args()
        cmd = {"token": args.token, "fn": "output_data_info"}
        conn = _connection()
        info = _query(cmd, conn)
        match info["storage"]:
            case Storage.Internal:
                cardinality = DataSchemaCardinality(info["cardinality"])
                schema_fields = info["schema"]
                fields = [
                    DataSchemaField(
                        field["label"],
                        field["dtype"],
                        field["required"],
                        field["nullable"],
                    )
                    for field in schema_fields
                ]
                schema = DataSchema(fields)

                super().__init__(schema=(cardinality, schema))
            case Storage.External:
                sources = info["sources"]
                super().__init__(sources=sources)

        self.__token = args.token
        self.__conn = conn

    def save(self):
        """Save data as output.

        Raises:
            ValueError: Requried value is missing.
            RuntimeError: Data could not be saved.
            ValueError: Error saving data.
        """
        if self.schema is not None:
            for field in self.schema:
                if field.required:
                    if field.label not in self._values:
                        raise ValueError(f"field {field.label} is required")
        elif self.sources is not None:
            for source in self.sources:
                if source.required:
                    if source.label not in self._values:
                        raise ValueError(f"source {source.label} is required")

        cmd = {"token": self.__token, "fn": "save_data"}
        resp = _query(cmd, self.__conn)
        if resp["status"] != "ok":
            raise RuntimeError(f"Could not save data: {resp["err"]}")

        properties = [
            {"label": prop.label, "type": prop.dtype, "value": prop.value}
            for prop in self._properties
        ]
        tags = [tag for tag in self._tags]
        data = {
            "token": self.__token,
            "properties": properties,
            "tags": tags,
            "values": self._values,
        }
        resp = _query(data, self.__conn)
        if resp["status"] == "err":
            raise ValueError(resp["err"])


def get_data() -> InputData:
    """Get the sample data.

    Returns:
        SampleData: Sample data.
    """
    args = Args()
    client = _connection()
    cmd = {"token": args.token, "fn": "get_data"}
    data = _query(cmd, client)
    storage = data["storage"]
    match storage:
        case Storage.External:
            if "data_paths" not in data:
                raise ValueError("invalid data")

            return InputData(
                args.token,
                data["tags"],
                data["properties"],
                data_source=data["data_paths"],
                conn=None,
            )
        case Storage.Internal:
            return InputData(
                args.token,
                data["tags"],
                data["properties"],
                data_source=None,
                conn=client,
            )

    raise ValueError("invalid storage")


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
