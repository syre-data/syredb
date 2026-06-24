# API functionality
from typing import Any, Optional, Iterable
from enum import StrEnum
import datetime as dt
import urllib.parse
import json
import requests
import uuid
from ._data import (
    QUANTITY_MAGNITUDE_KEY,
    QUANTITY_UNIT_KEY,
    Storage,
    ValueType,
    PropertyType,
    DataSchemaCardinality,
    DataSchemaField,
    DataSchema,
    Data as DataBase,
)


class Visibility(StrEnum):
    Private = "private"
    Public = "public"


class Data(DataBase):
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
        dtype = client._data_type(data_type)
        match dtype["storage"]:
            case Storage.Internal:
                cardinality = DataSchemaCardinality(dtype["schema"]["cardinality"])
                schema_fields = dtype["schema"]["fields"]
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
                sources = dtype["sources"]
                super().__init__(sources=sources)

        self.__data_type_id = dtype["id"]
        self._notes: list[tuple[dt.datetime, str]] = []

        self.origin = origin
        self.visibility = visiblility
        self.timestamp = timestamp

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

    def to_dict(self) -> dict[str, Any]:
        properties = [
            {"label": prop.label, "dtype": prop.dtype, "value": prop.value}
            for prop in self._properties
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
            raise RuntimeError("Invalid data")

        return data


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
