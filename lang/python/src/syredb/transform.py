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

__SYREDB_CONNECTION__ = None

LOCALHOST = "127.0.0.1"
QUANTITY_MAGNITUDE_KEY = "magnitude"
QUANTITY_UNIT_KEY = "unit"


class Storage(StrEnum):
    Internal = "internal"
    External = "external"


class ValueType(StrEnum):
    String = "string"
    Int = "int"
    UInt = "uint"
    Float = "float"
    Boolean = "boolean"
    Timestamp = "timestamp"


class PropertyType(StrEnum):
    String = "string"
    Int = "int"
    UInt = "uint"
    Float = "float"
    Boolean = "boolean"
    Timestamp = "timestamp"
    Quantity = "quantity"


class Cardinality(StrEnum):
    Single = "single"
    Multiple = "multiple"


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

    args = Args()
    if __SYREDB_CONNECTION__ is None:
        __SYREDB_CONNECTION__ = socket.create_connection((LOCALHOST, args.port))
    return __SYREDB_CONNECTION__


def _recv_exact(sock, n) -> bytes:
    buf = b""
    while len(buf) < n:
        chunk = sock.recv(n - len(buf))
        if not chunk:
            raise ConnectionError("socket closed")

        buf += chunk

    return buf


def _recv_message(sock) -> Any:
    header = _recv_exact(sock, 8)
    length = struct.unpack("<Q", header)[0]
    payload = _recv_exact(sock, length)
    return json.loads(payload)


def _query(content: dict[str, Any], sock: socket.socket) -> Any:
    data = json.dumps(content).encode()
    header = struct.pack("<Q", len(data))
    sock.sendall(header)
    sock.sendall(data)
    return _recv_message(sock)


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


class OutputData:
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
        self.__token = args.token
        self.__conn = _connection()

        cmd = {"token": self.__token, "fn": "output_data_info"}
        info = _query(cmd, self.__conn)
        match info["storage"]:
            case Storage.Internal:
                self.__schema = info["schema"]
                self.__cardinality = Cardinality(info["cardinality"])
                self.__sources = None
            case Storage.External:
                self.__schema = None
                self.__cardinality = None
                self.__sources = info["sources"]

        self._values: dict[str, Any] = {}
        self._properties: dict[str, Any] = {}
        self._tags: set[str] = set()

    def set_property(self, key: str, value: Any, dtype: Optional[PropertyType] = None):
        """Set a property value of the data.

        Args:
            key (str): Property key.
            value (Any): Property value.
            dtype (Optional[PropertyType], optional): Data type of the value.
            If `None`, the data type is inferred from `value`.
            Defaults to None.

        Notes:
            + `uint` data type (`dtype`) must be specified (i.e. can not be inferred)

        Raises:
            ValueError: Data type could not be inferred from value.
            ValueError: Value does not match provided data type.
        """
        if dtype is None:
            dtype = self._property_type_from_value(value)
            if dtype is None:
                raise ValueError(
                    "could not determine data type from value, please specify"
                )
        else:
            self._validate_value_as_property_type(value, dtype)

        self._properties[key] = {"type": dtype, "value": value}

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

    # TODO: Allow pandas and polars dataframes.
    def set_values(self, values: dict[str, Any]):
        if self.__schema is None:
            raise NotImplementedError(
                "data is not stored internally; use `.set_data_source` to set external data"
            )

        if isinstance(values, dict):
            self._set_values_dict(values)
        else:
            raise ValueError("invalid values")

    def _set_values_dict(self, values: dict[str, Any]):
        assert isinstance(self.__schema, list)

        height = None
        for key, val in values.items():
            field = None
            for s_field in self.__schema:
                if s_field["label"] == key:
                    field = s_field
                    break
            if field is None:
                raise ValueError(f"`{key}` is not a schema field")

            dtype = ValueType(field["dtype"])
            match self.__cardinality:
                case Cardinality.Single:
                    if not field["nullable"] or val is not None:
                        self._validate_value_as_data_type(val, dtype, key)
                    self._values[key] = val

                case Cardinality.Multiple:
                    if not isinstance(val, list):
                        raise ValueError(f"invalid value for `{key}`, expeted list")
                    if height is None:
                        height = len(val)
                    else:
                        if len(val) != height:
                            raise ValueError(f"invalid data length")
                    for v in val:
                        if not field["nullable"] or v is not None:
                            self._validate_value_as_data_type(v, dtype, key)

            self._values[key] = val

    @staticmethod
    def _property_type_from_value(value: Any) -> PropertyType | None:
        """Interpret data type from value.

        Args:
            value (Any): Value.

        Raises:
            ValueError: `value` is invalid.

        Returns:
            PropertyType | None: Property type of `value`, or None if it could not be determined.
        """
        if isinstance(value, str):
            return PropertyType.String
        elif isinstance(value, bool):
            return PropertyType.Boolean
        elif isinstance(value, dt.datetime):
            return PropertyType.Timestamp
        elif isinstance(value, float):
            return PropertyType.Float
        elif isinstance(value, int):
            return PropertyType.Int
        elif isinstance(value, dict):
            if QUANTITY_MAGNITUDE_KEY not in value or QUANTITY_UNIT_KEY not in value:
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
            return PropertyType.Quantity
        else:
            return None

    @staticmethod
    def _validate_value_as_property_type(value: Any, dtype: PropertyType):
        """Validate `value` as property type `dtype`.

        Args:
            value (Any): Value to validate.
            dtype (PropertyType): Target type.

        Raises:
            ValueError: `value` is not a valid `dtype`.
        """
        match dtype:
            case PropertyType.String:
                if not isinstance(value, str):
                    raise ValueError("invalid value for provided property type")
            case PropertyType.Boolean:
                if not isinstance(value, bool):
                    raise ValueError("invalid value for provided property type")
            case PropertyType.Timestamp:
                if not isinstance(value, dt.datetime):
                    raise ValueError("invalid value for provided property type")
            case PropertyType.Float:
                if not isinstance(value, float):
                    raise ValueError("invalid value for provided property type")
            case PropertyType.Int:
                if not isinstance(value, int):
                    raise ValueError("invalid value for provided property type")
            case PropertyType.UInt:
                if not isinstance(value, int) or value < 0:
                    raise ValueError("invalid value for provided property type")
            case PropertyType.Quantity:
                if (
                    QUANTITY_MAGNITUDE_KEY not in value
                    or QUANTITY_UNIT_KEY not in value
                ):
                    raise ValueError("invalid value for provided property type")
                if not isinstance(value[QUANTITY_UNIT_KEY], str):
                    raise ValueError("invalid value for provided property type")
                try:
                    magnitude = float(value[QUANTITY_MAGNITUDE_KEY])
                except ValueError:
                    raise ValueError("invalid value for provided property type")

                value = {
                    QUANTITY_MAGNITUDE_KEY: magnitude,
                    QUANTITY_UNIT_KEY: value[QUANTITY_UNIT_KEY],
                }

    @staticmethod
    def _validate_value_as_data_type(value: Any, dtype: ValueType, key: str):
        """Validates `value` is of type `dtype`. If valid, returns None.

        Args:
            value (Any): Value to validate.
            dtype (DType): Target data type.
            key (str): Field key, used for exception messages only.

        Raises:
            ValueError: `value` is not valid.
        """
        match dtype:
            case ValueType.String:
                if not isinstance(value, str):
                    raise ValueError(f"invalid value for `{key}`, expeted {dtype}")
            case ValueType.Int:
                if not isinstance(value, int):
                    raise ValueError(f"invalid value for `{key}`, expeted {dtype}")
            case ValueType.UInt:
                if not isinstance(value, int):
                    raise ValueError(f"invalid value for `{key}`, expeted {dtype}")
                if value < 0:
                    raise ValueError(f"invalid value for `{key}`, must be non-negative")
            case ValueType.Float:
                if not isinstance(value, float) and not isinstance(value, int):
                    raise ValueError(f"invalid value for `{key}`, expeted {dtype}")
            case ValueType.Boolean:
                if not isinstance(value, bool):
                    raise ValueError(f"invalid value for `{key}`, expeted {dtype}")
            case ValueType.Timestamp:
                if not isinstance(value, dt.datetime):
                    raise ValueError(f"invalid value for `{key}`, expeted {dtype}")

    def set_data_source(self, key: str, source: Any):
        if self.__sources is None:
            raise NotImplementedError(
                "data is not stored externally; use `.set_values` to set data values"
            )

        raise NotImplementedError("TODO")

    def save(self):
        """Save data as output.

        Raises:
            ValueError: Requried value is missing.
            RuntimeError: Data could not be saved.
            ValueError: Error saving data.
        """
        if self.__schema is not None:
            for field in self.__schema:
                if field["required"]:
                    if field["label"] not in self._values:
                        raise ValueError(f"field {field["label"]} is required")
        elif self.__sources is not None:
            for source in self.__sources:
                if source["required"]:
                    if source["label"] not in self._values:
                        raise ValueError(f"source {source["label"]} is required")
        else:
            raise RuntimeError("invalid storage type")

        cmd = {"token": self.__token, "fn": "save_data"}
        resp = _query(cmd, self.__conn)
        if resp["status"] != "ok":
            raise RuntimeError(f"could not save data: {resp["err"]}")

        properties = [
            {"key": key, "type": val["type"], "value": val["value"]}
            for key, val in self._properties.items()
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
