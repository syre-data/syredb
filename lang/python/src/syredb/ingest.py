from typing import Any
import sys
from enum import StrEnum
from collections import namedtuple
import socket
from ._socket import _query, LOCALHOST
from ._data import (
    ValueType,
    DataSchemaField,
    DataSchema,
    Data as DataBase,
)

__SYREDB_CONNECTION__ = None


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

    if __SYREDB_CONNECTION__ is None:
        args = Args()
        __SYREDB_CONNECTION__ = socket.create_connection((LOCALHOST, args.port))
    return __SYREDB_CONNECTION__


FileInfo = namedtuple("FileInfo", ["path", "filename"])


class DataSource:
    def __init__(self, cardinality: Cardinality, source: FileInfo | list[FileInfo]):
        if (cardinality is Cardinality.Single and not isinstance(source, FileInfo)) or (
            cardinality is Cardinality.Multiple and not isinstance(source, list)
        ):
            raise ValueError("`cardinality` and `source` are incompatible")

        self.__cardinality = cardinality
        self.__source = source

    @property
    def cardinality(self) -> Cardinality:
        return self.__cardinality

    @property
    def source(self) -> FileInfo:
        if self.__cardinality is not Cardinality.Single:
            raise NotImplementedError(
                "`source` is not implemented for this cardinality, use `.sources()` instead"
            )
        assert isinstance(self.__source, FileInfo)
        return self.__source

    @property
    def sources(self) -> list[FileInfo]:
        if self.__cardinality is not Cardinality.Multiple:
            raise NotImplementedError(
                "`sources` is not implemented for this cardinality, use `.source()` instead"
            )
        assert isinstance(self.__source, list)
        return self.__source


class Data(DataBase):
    def __init__(self, token: str):
        super().__init__()
        self.__token = token

    def save(self):
        raise NotImplementedError("TODO")


def get_sources() -> dict[str, DataSource]:
    args = Args()
    client = _connection()
    cmd = {"token": args.token, "fn": "get_sources"}
    sources = _query(cmd, client)
    return sources
