from typing import Any, Optional, Iterable, Literal
from enum import StrEnum
import datetime as dt

QUANTITY_MAGNITUDE_KEY = "magnitude"
QUANTITY_UNIT_KEY = "unit"


class Storage(StrEnum):
    Internal = "internal"
    External = "external"


class DataSchemaCardinality(StrEnum):
    Single = "single"
    Multiple = "multiple"


class DataSourceCardinality(StrEnum):
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


class Property:
    def __init__(self, label: str, dtype: PropertyType, value: Optional[Any] = None):
        if value is not None:
            self.validate_value_as_dtype(value, dtype)

        self.__label = label
        self.__dtype = dtype
        self._value = value

    @property
    def label(self) -> str:
        return self.__label

    @property
    def dtype(self) -> PropertyType:
        return self.__dtype

    @property
    def value(self) -> Any:
        return self._value

    @value.setter
    def value(self, value: Any):
        self.validate_value_as_dtype(value, self.__dtype)
        self._value = value

    @staticmethod
    def dtype_from_value(value: Any) -> PropertyType | None:
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
                    "Could not determine data type from value, please specify"
                )
            if not isinstance(value[QUANTITY_UNIT_KEY], str):
                raise ValueError("Invalid quanitity data")
            try:
                magnitude = float(value[QUANTITY_MAGNITUDE_KEY])
            except ValueError:
                raise ValueError("Invalid value for provided data type")

            value = {
                QUANTITY_MAGNITUDE_KEY: magnitude,
                QUANTITY_UNIT_KEY: value[QUANTITY_UNIT_KEY],
            }
            return PropertyType.Quantity
        else:
            return None

    @staticmethod
    def validate_value_as_dtype(value: Any, dtype: PropertyType):
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
                    raise ValueError("Invalid value for provided property type")
            case PropertyType.Boolean:
                if not isinstance(value, bool):
                    raise ValueError("Invalid value for provided property type")
            case PropertyType.Timestamp:
                if not isinstance(value, dt.datetime):
                    raise ValueError("Invalid value for provided property type")
            case PropertyType.Float:
                if not isinstance(value, float):
                    raise ValueError("Invalid value for provided property type")
            case PropertyType.Int:
                if not isinstance(value, int):
                    raise ValueError("Invalid value for provided property type")
            case PropertyType.UInt:
                if not isinstance(value, int) or value < 0:
                    raise ValueError("Invalid value for provided property type")
            case PropertyType.Quantity:
                if (
                    QUANTITY_MAGNITUDE_KEY not in value
                    or QUANTITY_UNIT_KEY not in value
                ):
                    raise ValueError("Invalid value for provided property type")
                if not isinstance(value[QUANTITY_UNIT_KEY], str):
                    raise ValueError("Invalid value for provided property type")
                try:
                    magnitude = float(value[QUANTITY_MAGNITUDE_KEY])
                except ValueError:
                    raise ValueError("Invalid value for provided property type")

                value = {
                    QUANTITY_MAGNITUDE_KEY: magnitude,
                    QUANTITY_UNIT_KEY: value[QUANTITY_UNIT_KEY],
                }


class PropertyList:
    def __init__(self):
        self._inner: list[Property] = []

    def set(self, key: str, value: Any, dtype: Optional[PropertyType] = None):
        """Set a property value.

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
            dtype = Property.dtype_from_value(value)
            if dtype is None:
                raise ValueError(
                    f"Could not determine data type of `{key}` from value; specify using `dtype`"
                )

        prop = None
        for p in self._inner:
            if p.label == key:
                prop = p
                break

        if prop is None:
            prop = Property(key, dtype, value)
            self._inner.append(prop)
        else:
            prop.value = value

    def __getitem__(self, key: str) -> Property:
        for p in self._inner:
            if p.label == key:
                prop = p
                return p

        raise KeyError(f"Property with key `{key}` not found")

    def __setitem__(self, key, value):
        dtype = Property.dtype_from_value(value)
        if dtype is None:
            raise ValueError(
                f"Could not determine data type of `{key}` from value; use `.set()` instead"
            )

        prop = None
        for p in self._inner:
            if p.label == key:
                prop = p
                break

        if prop is None:
            prop = Property(key, dtype, value)
            self._inner.append(prop)
        else:
            prop.value = value

    def __iter__(self):
        for prop in self._inner:
            yield prop


class DataSchemaField:
    def __init__(
        self,
        label: str,
        dtype: ValueType,
        required: bool,
        nullable: bool,
    ):
        self.__label = label
        self.__dtype = dtype
        self.__required = required
        self.__nullable = nullable

    @property
    def label(self) -> str:
        return self.__label

    @property
    def dtype(self) -> ValueType:
        return self.__dtype

    @property
    def required(self) -> bool:
        return self.__required

    @property
    def nullable(self) -> bool:
        return self.__nullable


class DataSchema:
    def __init__(
        self,
        fields: list[DataSchemaField],
    ):
        labels = [field.label for field in fields]
        labels.sort()
        lbl_len = len(labels)
        for idx, lbl in enumerate(labels):
            if idx + 1 > lbl_len - 1:
                break

            if labels[idx + 1] == lbl:
                raise ValueError(f"Field `{lbl}` is duplicated")

        self._fields = fields

    def __getitem__(self, key: str) -> DataSchemaField:
        for field in self._fields:
            if field.label == key:
                return field

        raise KeyError(f"Field `{key}` not found")

    def __iter__(self):
        for field in self._fields:
            yield field


class DataSource:
    def __init__(
        self,
        label: str,
        required: bool,
        cardinality: DataSourceCardinality,
        ext_filter: list[str],
    ):
        self.__label = label
        self.__required = required
        self.__cardinality = cardinality
        self.__ext_filter = ext_filter

    @property
    def label(self) -> str:
        return self.__label

    @property
    def required(self) -> bool:
        return self.__required

    @property
    def cardinality(self) -> DataSourceCardinality:
        return self.__cardinality

    @property
    def ext_filter(self) -> list[str]:
        return self.__ext_filter


class DataSourceList:
    def __init__(
        self,
        sources: list[DataSource],
    ):
        labels = [source.label for source in sources]
        labels.sort()
        lbl_len = len(labels)
        for idx, lbl in enumerate(labels):
            if idx >= lbl_len:
                break

            if labels[idx + 1] == lbl:
                raise ValueError(f"Source `{lbl}` is duplicated")

        self._sources = sources

    def __getitem__(self, key: str) -> DataSource:
        for source in self._sources:
            if source.label == key:
                return source

        raise KeyError(f"Source `{key}` not found")

    def __iter__(self):
        for source in self._sources:
            yield source


class Data:
    def __init__(
        self,
        schema: Optional[tuple[DataSchemaCardinality, DataSchema]] = None,
        sources: Optional[DataSourceList] = None,
    ):
        if schema is None and sources is None:
            raise ValueError("One of `schema` or `sources` must be provided")
        if schema is not None and sources is not None:
            raise ValueError("Only one of `schema` and `sources` can be provided")

        self.__cardinality: DataSchemaCardinality | None = None
        self.__schema: DataSchema | None = None
        self.__sources: DataSourceList | None = None
        if schema is not None:
            cardinaltiy, fields = schema
            self.__cardinality = cardinaltiy
            self.__schema = fields
        elif sources is not None:
            self.__sources = sources

        self._tags: set[str] = set()
        self._properties: PropertyList = PropertyList()
        self._values: dict[str, Any] = {}

    @property
    def schema(self) -> DataSchema | None:
        return self.__schema

    @property
    def sources(self) -> DataSourceList | None:
        return self.__sources

    @property
    def properties(self) -> PropertyList:
        return self._properties

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
                "set_values can not be called on data types with external storage, use set_data_source"
            )

        if isinstance(values, dict):
            self._set_values_dict(values)
        else:
            raise ValueError("invalid values")

    def _set_values_dict(self, values: dict[str, Any]):
        assert self.__schema is not None

        height = None
        for key, val in values.items():
            field = None
            for s_field in self.__schema:
                if s_field.label == key:
                    field = s_field
                    break
            if field is None:
                raise ValueError(f"`{key}` is not a schema field")

            dtype = ValueType(field.dtype)
            match self.__cardinality:
                case DataSchemaCardinality.Single:
                    if not field.nullable or val is not None:
                        self.validate_value_as_dtype(val, dtype, key)
                    self._values[key] = val

                case DataSchemaCardinality.Multiple:
                    if not isinstance(val, list):
                        raise ValueError(f"invalid value for `{key}`, expected list")
                    if height is None:
                        height = len(val)
                    else:
                        if len(val) != height:
                            raise ValueError(f"invalid data length")
                    for v in val:
                        if not field.nullable or v is not None:
                            self.validate_value_as_dtype(v, dtype, key)

            self._values[key] = val

    def set_data_source(self, key: str, source: Any):
        if self.__sources is None:
            raise NotImplementedError(
                "set_data_source can not be called on data types with internal storage, "
                + "use set_values"
            )

        raise NotImplementedError("TODO")

    @staticmethod
    def validate_value_as_dtype(value: Any, dtype: ValueType, key: str):
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
