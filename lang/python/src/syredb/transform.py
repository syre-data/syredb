from typing import Any, Optional, Iterable
import os
import sys
from enum import StrEnum
import uuid
import json
import datetime as dt
import numpy
import pandas

QUANTITY_MAGNITUDE_KEY = "magnitude"
QUANTITY_UNIT_KEY = "unit"


class Storage(StrEnum):
    Internal = "internal"
    External = "external"


class DType(StrEnum):
    String = "string"
    Int = "int"
    UInt = "uint"
    Float = "float"
    Boolean = "boolean"
    Timestamp = "timestamp"
    Quantity = "quantity"


class Args:
    def __init__(self):
        self._token = sys.argv[1]
        self._data_path = sys.argv[2]

    @property
    def token(self) -> str:
        return self._token

    @property
    def path(self) -> str:
        return self._data_path


class InputData:
    """Input data.

    Example:
        ```
        from syredb.transform import get_data, OutputData

        input = get_data()
        df = input.as_pandas() # only valid for internally stored data
        df_avg = df.sum() / input.properties["sample_count"]

        output = OutputData()
        output.set_data(df_avg)
        output.save()
        ```
    """

    def __init__(
        self,
        token: str,
        storage: Storage,
        data_paths: str | dict[str, str | list[str]],
        tags: set,
        properties: dict[str, Any],
    ):
        match storage:
            case Storage.Internal:
                if not isinstance(data_paths, str):
                    raise ValueError("`data_paths` and `storage` are incompatible")
                self.__data_sources = None
                self.__data_path = data_paths
            case Storage.External:
                if not isinstance(data_paths, dict):
                    raise ValueError("`data_paths` and `storage` are incompatible")
                self.__data_sources = data_paths
                self.__data_path = None

        self.__token = token
        self._tags = set()
        self._properties = {}

    @property
    def data_sources(self) -> dict[str, str | list[str]]:
        """Get the paths of the data sources.

        Raises:
            KeyError: Data's storage is not external.

        Returns:
            dict[str, str | list[str]: Paths to source file(s) keyed by label.
        """
        if self.__data_sources is None:
            raise KeyError("data sources are only available for externally stored data")

        return self.__data_sources

    @property
    def data_path(self) -> str:
        """Get the path to the `.arrow` file storing the data.

        Raises:
            KeyError: Data's storage is not internal.

        Returns:
            str: Path to the data file.
        """
        if self.__data_path is None:
            raise KeyError("data path is only available for internally stored data")

        return self.__data_path

    @property
    def properties(self) -> dict[str, Any]:
        """Properties associated with the sample data.
        These include those inherited from the sample and groups.
        """
        return self._properties

    def as_pandas(self) -> pandas.DataFrame:
        if self.__data_path is None:
            raise NotImplementedError(
                "data type can not be represented as a dataframe; use `.data_sources` to load data yourself"
            )

        return pandas.read_feather(self.data_path)


class OutputData:
    """Output data.

    Example:
        ```
        from syredb.transform import get_data, OutputData

        input = get_data()
        df = input.as_pandas() # only valid for internally stored data
        df_avg = df.sum() / input.properties["sample_count"]

        output = OutputData()
        output.set_data(df_avg)
        output.set_property("count", 10)
        output.add_tag("summary")
        output.save()
        ```
    """

    def __init__(self):
        args = Args()
        self._properties = {}
        self._data = None
        self._tags = set()

    def set_property(self, key: str, value: Any, dtype: Optional[DType] = None):
        """Set a property value of the data.

        Args:
            key (str): Property key.
            value (Any): Property value.
            dtype (Optional[DType], optional): Data type of the value.
            If `None`, the data type is inferred from `value`.
            Defaults to None.

        Notes:
            + `uint` data type must be specified (i.e. can not be inferred)

        Raises:
            ValueError: Data type could not be inferred from value.
            ValueError: Value does not match provided data type.
        """
        if dtype is None:
            if isinstance(value, str):
                dtype = DType.String
            elif isinstance(value, bool):
                dtype = DType.Boolean
            elif isinstance(value, dt.datetime):
                dtype = DType.Timestamp
            elif isinstance(value, float):
                dtype = DType.Float
            elif isinstance(value, int):
                dtype = DType.Int
            elif isinstance(value, dict):
                if (
                    QUANTITY_MAGNITUDE_KEY not in value
                    or QUANTITY_UNIT_KEY not in value
                ):
                    raise ValueError(
                        "could not determine data type from value, please specify"
                    )
                if not isinstance(value[QUANTITY_UNIT_KEY], str):
                    raise ValueError("invalid quanitity data")
                try:
                    magnitude = float(value[QUANTITY_MAGNITUDE_KEY])
                except ValueError:
                    raise ValueError("invalid value for provided data type")

                value = {
                    QUANTITY_MAGNITUDE_KEY: magnitude,
                    QUANTITY_UNIT_KEY: value[QUANTITY_UNIT_KEY],
                }
                dtype = DType.Quantity
            else:
                raise ValueError(
                    "could not determine data type from value, please specify"
                )
        else:
            match dtype:
                case DType.String:
                    if not isinstance(value, str):
                        raise ValueError("invalid value for provided data type")
                case DType.Boolean:
                    if not isinstance(value, bool):
                        raise ValueError("invalid value for provided data type")
                case DType.Timestamp:
                    if not isinstance(value, dt.datetime):
                        raise ValueError("invalid value for provided data type")
                case DType.Float:
                    if not isinstance(value, float):
                        raise ValueError("invalid value for provided data type")
                case DType.Int:
                    if not isinstance(value, int):
                        raise ValueError("invalid value for provided data type")
                case DType.UInt:
                    if not isinstance(value, int) or value < 0:
                        raise ValueError("invalid value for provided data type")
                case DType.Quantity:
                    if (
                        QUANTITY_MAGNITUDE_KEY not in value
                        or QUANTITY_UNIT_KEY not in value
                    ):
                        raise ValueError("invalid value for provided data type")
                    if not isinstance(value[QUANTITY_UNIT_KEY], str):
                        raise ValueError("invalid value for provided data type")
                    try:
                        magnitude = float(value[QUANTITY_MAGNITUDE_KEY])
                    except ValueError:
                        raise ValueError("invalid value for provided data type")

                    value = {
                        QUANTITY_MAGNITUDE_KEY: magnitude,
                        QUANTITY_UNIT_KEY: value[QUANTITY_UNIT_KEY],
                    }

        self._properties[key] = {"key": key, "type": dtype, "value": value}

    def add_tag(self, tag: str):
        """Add tag to the data.

        Args:
            tag (str): Tag.
        """
        self._tags.add(tag)

    def add_tags(self, tags: Iterable[str]):
        """Add multiple tags to the data.

        Args:
            tags (Iterable[str]): Tags to add.
        """
        self._tags.update(tags)

    def save(self):
        raise NotImplementedError("TODO")


def get_data() -> InputData:
    """Get the sample data.

    Returns:
        SampleData: Sample data.
    """
    args = Args()
    with open(args.path) as f:
        data = json.load(f)

    match args.source_storage:
        case Storage.Internal:
            data_paths = data["data_path"]
        case Storage.External:
            data_paths = data["data_sources"]

    return Data(
        args.token,
        args.source_storage,
        data_paths,
        data["tags"],
        data["properties"],
        args.destination_storage,
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
