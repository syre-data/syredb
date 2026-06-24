from typing import Any
import sys
from enum import StrEnum
import socket
from ._socket import _query, LOCALHOST
from ._data import (
    ValueType,
    validate_value_as_data_type,
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


class DataSource:
    def __init__(
        self, 
        label: str, 
        required: bool, 
        cardinality: Cardinality
    ):
        self.__label = label
        self.__required = required
        self.__cardinality = cardinality
        
    @property
    def required(self)-> bool:
        return self.__required
    
    @property
    def cardinality(self) -> Cardinality:
        return self.__cardinality

class Data(DataBase):
    def __init__(self, token: str):
        super().__init__()
        self.__token = token
        
        
    def set_values(self):
        
    def save(self):
    

def get_sources() -> dict[str, DataSource]:
    