# API functionality
from typing import Any, Optional, Iterable
from enum import StrEnum
import datetime as dt
import urllib.parse
import json
import requests
import uuid

QUANTITY_MAGNITUDE_KEY = "magnitude"
QUANTITY_UNIT_KEY = "unit"


class Visibility(StrEnum):
    Private = "private"
    Public = "public"


class Storage(StrEnum):
    Internal = "internal"
    External = "external"


class Cardinality(StrEnum):
    Single = "single"
    Multiple = "multiple"


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


class Data:
    """Data.

    Example:
        ```py
        from syredb.api import Data, Client

        df = get_values() # values collected from experiment
        owner = get_user_credentials() # user id of data owner

        client = Client(host="127.1.1.1", token="MY_SECRET_TOKEN")
        data = Data(client, "my_data_type") # initialize data as the given type
        data.add_tag("summary")
        data.set_property("count", 10)
        data.set_values(df_avg) # only valid for internally stored data

        client.insert(data)
        ```
    """

    def __init__(
        self,
        client: "Client",
        data_type: uuid.UUID | str,
        origin: uuid.UUID | str,
        visiblility: Visibility = Visibility.Private,
        timestamp: dt.datetime = dt.datetime.now(tz=dt.timezone.utc),
    ):
        self.origin = origin
        self.visibility = visiblility
        self.timestamp = timestamp

        dtype = client._data_type(data_type)
        self.__data_type_id = dtype["id"]
        match dtype["storage"]:
            case Storage.Internal:
                self.__schema = dtype["schema"]["fields"]
                self.__cardinality = Cardinality(dtype["schema"]["cardinality"])
                self.__sources = None
            case Storage.External:
                self.__schema = None
                self.__cardinality = None
                self.__sources = dtype["sources"]

        self._values: dict[str, Any] = {}
        self._properties: dict[str, Any] = {}
        self._tags: set[str] = set()
        self._notes: list[tuple[dt.datetime, str]] = []

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

    def add_note(
        self, content: str, timestamp: dt.datetime = dt.datetime.now(tz=dt.timezone.utc)
    ):
        """Add a note to the data.

        Args:
            content (str): Note content.
            timestamp (dt.datetime): Timestamp assosciated with the note.
            Defaults to dt.datetime.now(tz=dt.timezone.utc).
        """
        self._notes.append((timestamp, content))

    # TODO: Allow pandas and polars dataframes.
    def set_values(self, values: dict[str, Any]):
        if self.__schema is None:
            raise NotImplementedError(
                "set_values can not be called on data types with external storage, use set_data_source"
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

    def set_data_source(self, key: str, source: Any):
        if self.__sources is None:
            raise NotImplementedError(
                "set_data_source can not be called on data types with internal storage, "
                + "use set_values"
            )

        raise NotImplementedError("TODO")

    def to_dict(self) -> dict[str, Any]:
        properties = [
            {"key": key, "dtype": vals["type"], "value": vals["value"]}
            for key, vals in self._properties.items()
        ]
        notes = [
            {"timestamp": timestamp.isoformat(), "content": content}
            for (timestamp, content) in self._notes
        ]
        data = {
            "id": self.__data_type_id,
            "origin": self.origin,
            "visibility": self.visibility,
            "timestamp": self.timestamp.isoformat(),
            "properties": properties,
            "tags": list(self._tags),
            "notes": notes,
        }

        if self.__sources is not None:
            data["storage"] = Storage.External
            data["sources"] = [
                {"label": label, "sources": values}
                for label, values in self._values.items()
            ]
        elif self.__schema is not None:
            data["storage"] = Storage.Internal
            data["fields"] = [
                {"label": label, "values": values}
                for label, values in self._values.items()
            ]
        else:
            raise RuntimeError("invalid data")

        return data

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


class Client:
    def __init__(
        self,
        host: str,
        email: str,
        password: str,
        expiration: dt.datetime,
    ):
        self.host = host
        self._authenticate_user(email, password, expiration)

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        url = urllib.parse.urljoin(self.url, "deactivate")
        data = {"token": self.__token}
        res = requests.post(url, data=data)
        res.raise_for_status()

    @property
    def url(self) -> str:
        # TODO: https
        return f"http://{self.host}/api/client/"

    def _authenticate_user(
        self,
        email: str,
        password: str,
        expiration: dt.datetime,
    ):
        url = urllib.parse.urljoin(self.url, "authenticate")
        data = {
            "email": email,
            "password": password,
            "expiration": expiration.isoformat(),
        }
        res = requests.post(url, data=data)
        res.raise_for_status()

        self.__token = res.text

    def _data_type(self, data_type: uuid.UUID | str) -> dict[str, Any]:
        url = urllib.parse.urljoin(self.url, "data-type")
        data = {
            "token": self.__token,
            "data_type": data_type,
        }
        res = requests.get(url, params=data)
        res.raise_for_status()
        return res.json()

    def insert(
        self,
        data: Data,
    ):
        if len(data._values) == 0:
            raise ValueError("data values can not be empty")

        url = urllib.parse.urljoin(self.url, "data/create")
        fields = {
            "token": self.__token,
            "data": json.dumps(data.to_dict()),
        }
        res = requests.post(url, data=fields)
        res.raise_for_status()
